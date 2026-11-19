package stats

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSave(t *testing.T) {
	url := os.Getenv("STATS_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set STATS_TEST_DATABASE_URL to run the PostgreSQL test")
	}
	ctx := t.Context()
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = 1
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	// Keep the migrated tables in this connection's temporary schema.
	if _, err := db.Exec(ctx, "CREATE TEMP TABLE init_session (); SET search_path TO pg_temp"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"000001_init_schema", "000002_user_streaks"} {
		sql, err := os.ReadFile("../../postgres/migrations/" + name + ".up.sql")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(ctx, string(sql)); err != nil {
			t.Fatal(err)
		}
	}
	start := time.Date(2026, 1, 1, 23, 30, 0, 0, time.UTC)
	events := []QuestionAttemptEvent{
		{AnswerID: 1, UserID: 7, OccurredAt: start, IsCorrect: true, FirstSolve: true, XP: 20},
		{AnswerID: 2, UserID: 7, OccurredAt: start.In(time.FixedZone("UTC+3", 3*3600)), IsCorrect: true},
		{AnswerID: 3, UserID: 7, OccurredAt: start.Add(time.Hour), IsCorrect: true, FirstSolve: true, XP: 10},
		{AnswerID: 4, UserID: 7, OccurredAt: start.Add(72 * time.Hour)},
		{AnswerID: 5, UserID: 7, OccurredAt: start.Add(72 * time.Hour), IsCorrect: true},
	}
	service := NewService(NewRepository(db))
	for _, event := range events {
		if saved, err := service.Save(ctx, event); err != nil || !saved {
			t.Fatalf("save answer %d: saved=%t, err=%v", event.AnswerID, saved, err)
		}
	}
	if saved, err := service.Save(ctx, events[0]); err != nil || saved {
		t.Fatalf("redelivery: saved=%t, err=%v", saved, err)
	}
	totals, err := service.GetStats(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if totals.UserID != 7 || totals.XP != 30 || totals.TotalAnswers != 5 || totals.CorrectAnswers != 4 ||
		totals.UniqueSolved != 2 || totals.ActiveDays != 3 || totals.CurrentStreak != 1 || totals.MaxStreak != 2 ||
		totals.CurrentCorrectStreak != 1 || totals.BestCorrectStreak != 3 ||
		totals.LastSolvedOn == nil || totals.LastSolvedOn.Format(time.DateOnly) != "2026-01-04" {
		t.Fatalf("unexpected stats: %+v", totals)
	}

	for _, check := range []struct {
		name string
		get  func(context.Context, int64) (int64, error)
		want int64
	}{
		{"XP", service.GetXP, 30},
		{"current streak", service.GetCurrentStreak, 1},
		{"current correct streak", service.GetCurrentCorrectStreak, 1},
	} {
		if got, err := check.get(ctx, 7); err != nil || got != check.want {
			t.Errorf("%s = %d, err = %v, want %d", check.name, got, err, check.want)
		}
		if _, err := check.get(ctx, 8); !errors.Is(err, pgx.ErrNoRows) {
			t.Errorf("%s for missing user: %v, want ErrNoRows", check.name, err)
		}
	}
	if _, err := service.GetStats(ctx, 8); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("stats for missing user: %v, want ErrNoRows", err)
	}
}
