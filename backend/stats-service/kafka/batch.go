package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/calmarie/scoreup_train/backend/stats-service/internal/stats"
	"github.com/twmb/franz-go/pkg/kgo"
	"golang.org/x/sync/errgroup"
)

const maxParallelUsers = 4

// processBatch limits concurrent saves because each holds a database connection.
// Four is a starting point. Tune it using consumer lag and connection pool wait times.
func processBatch(
	ctx context.Context,
	logger *slog.Logger,
	service stats.StatsService,
	records []*kgo.Record,
	commit func(context.Context, ...*kgo.Record) error,
) error {
	if len(records) == 0 {
		return nil
	}
	byUser, err := groupByUser(records)
	if err != nil {
		return err
	}

	saves, saveCtx := errgroup.WithContext(ctx)
	saves.SetLimit(maxParallelUsers)
	for _, attempts := range byUser {
		saves.Go(func() error {
			return saveUserAttempts(saveCtx, logger, service, attempts)
		})
	}
	if err := saves.Wait(); err != nil {
		return err
	}
	if err := commit(ctx, records...); err != nil {
		return fmt.Errorf("commit Kafka batch: %w", err)
	}
	return nil
}

// groupByUser keeps each user's attempts in Kafka order for streak calculations.
func groupByUser(records []*kgo.Record) (map[int64][]QuestionAttemptEvent, error) {
	byUser := make(map[int64][]QuestionAttemptEvent)
	for _, record := range records {
		var event QuestionAttemptEvent
		if err := json.Unmarshal(record.Value, &event); err != nil {
			return nil, fmt.Errorf("decode Kafka event at %s/%d/%d: %w", record.Topic, record.Partition, record.Offset, err)
		}
		byUser[event.UserID] = append(byUser[event.UserID], event)
	}
	return byUser, nil
}

// saveUserAttempts saves one user's answers sequentially so streaks follow Kafka order.
func saveUserAttempts(ctx context.Context, logger *slog.Logger, service stats.StatsService, attempts []QuestionAttemptEvent) error {
	for _, event := range attempts {
		saved, err := service.Save(ctx, event)
		if err != nil {
			return fmt.Errorf("save stats for answer %d: %w", event.AnswerID, err)
		}
		if saved {
			logger.Info("question attempt saved",
				"answer_id", event.AnswerID,
				"user_id", event.UserID,
				"question_id", event.QuestionID,
				"topic", event.Topic,
				"difficulty", event.Difficulty,
				"is_correct", event.IsCorrect,
				"xp", event.XP,
				"occurred_at", event.OccurredAt,
				"first_solve", event.FirstSolve,
			)
		}
	}
	return nil
}
