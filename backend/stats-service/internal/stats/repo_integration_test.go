package stats

import (
	"os"
	"testing"
	"time"

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
	var xp, total, correct, unique, days, streak, maxStreak, consecutive, best int
	var lastSolved time.Time
	err = db.QueryRow(ctx, `SELECT xp, total_answers, correct_answers, unique_solved, active_days,
		current_streak, max_streak, current_correct_streak, best_correct_streak, last_solved_on
		FROM user_stats WHERE user_id = 7`).Scan(&xp, &total, &correct, &unique, &days,
		&streak, &maxStreak, &consecutive, &best, &lastSolved)
	if err != nil {
		t.Fatal(err)
	}
	if xp != 30 || total != 5 || correct != 4 || unique != 2 || days != 3 ||
		streak != 1 || maxStreak != 2 || consecutive != 1 || best != 3 || lastSolved.Format(time.DateOnly) != "2026-01-04" {
		t.Fatalf("xp=%d total=%d correct=%d unique=%d days=%d streak=%d max=%d consecutive=%d best=%d last=%s",
			xp, total, correct, unique, days, streak, maxStreak, consecutive, best, lastSolved)
	}
}
