package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/twmb/franz-go/pkg/kgo"
)

type saveFunc func(context.Context, QuestionAttemptEvent) (bool, error)

func (f saveFunc) Save(ctx context.Context, event QuestionAttemptEvent) (bool, error) {
	return f(ctx, event)
}

func TestBatchSavesUsersConcurrentlyAndInOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var records []*kgo.Record
		for answerID := 1; answerID <= 12; answerID++ {
			userID := (answerID-1)%6 + 1
			value := fmt.Sprintf(`{"user_id":%d,"answer_id":%d}`, userID, answerID)
			records = append(records, &kgo.Record{Value: []byte(value)})
		}

		started := make(chan QuestionAttemptEvent, len(records))
		release := make(chan struct{})
		var completed atomic.Int32
		save := saveFunc(func(_ context.Context, event QuestionAttemptEvent) (bool, error) {
			started <- event
			<-release
			completed.Add(1)
			return event.AnswerID != 1, nil // One attempt was already saved.
		})
		var commits int
		commit := func(ctx context.Context, batch ...*kgo.Record) error {
			commits++
			if n := int(completed.Load()); n != len(records) {
				t.Errorf("commit after %d of %d saves", n, len(records))
			}
			if !slices.Equal(records, batch) {
				t.Error("commit did not cover the whole batch")
			}
			return ctx.Err()
		}
		done := make(chan error, 1)
		go func() { done <- processBatch(t.Context(), slog.New(slog.DiscardHandler), save, records, commit) }()
		synctest.Wait()
		if len(started) != 4 || commits != 0 {
			t.Errorf("while saves are blocked: %d started, %d commits; want 4 and 0", len(started), commits)
		}
		close(release)
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		if commits != 1 {
			t.Errorf("commits = %d, want 1", commits)
		}
		close(started)
		byUser := make(map[int64][]int64)
		for event := range started {
			byUser[event.UserID] = append(byUser[event.UserID], event.AnswerID)
		}
		for userID := int64(1); userID <= 6; userID++ {
			if got := byUser[userID]; !slices.Equal(got, []int64{userID, userID + 6}) {
				t.Errorf("user %d: answers %v", userID, got)
			}
		}
	})
}

func TestFailedBatchSkipsCommit(t *testing.T) {
	saveErr := errors.New("save failed")
	for _, tc := range []struct {
		name        string
		secondValue string
		wantSaves   int
	}{
		{"invalid JSON", "{", 0},
		{"save error", `{"user_id":1,"answer_id":2}`, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			records := []*kgo.Record{
				{Value: []byte(`{"user_id":1,"answer_id":1}`)},
				{Value: []byte(tc.secondValue)},
			}
			saves := 0
			save := saveFunc(func(_ context.Context, event QuestionAttemptEvent) (bool, error) {
				saves++
				if event.AnswerID == 2 {
					return false, saveErr
				}
				return true, nil
			})
			commit := func(context.Context, ...*kgo.Record) error {
				t.Error("committed a failed batch")
				return nil
			}
			err := processBatch(t.Context(), slog.New(slog.DiscardHandler), save, records, commit)
			if err == nil || saves != tc.wantSaves {
				t.Errorf("saves = %d, want %d; err = %v", saves, tc.wantSaves, err)
			}
		})
	}
}
