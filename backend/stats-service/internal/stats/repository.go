package stats

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StatsRepository interface {
	Update(ctx context.Context, event QuestionAttemptEvent, day time.Time, apply func(*UserStats)) (bool, error)
	GetStats(ctx context.Context, userID int64) (UserStatsDTO, error)
	GetXP(ctx context.Context, userID int64) (int64, error)
	GetCurrentStreak(ctx context.Context, userID int64) (int64, error)
	GetCurrentCorrectStreak(ctx context.Context, userID int64) (int64, error)
	GetDailyStats(ctx context.Context, userID int64, day time.Time) (UserDailyStats, error)
	GetDailyStatsRange(ctx context.Context, userID int64, from, through time.Time) ([]UserDailyStats, error)
	GetTopicStats(ctx context.Context, userID int64, topic string) (UserTopicStats, error)
	GetAllTopicStats(ctx context.Context, userID int64) ([]UserTopicStats, error)
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Update applies each answer once across the total, daily, and topic stats.
func (r *Repository) Update(ctx context.Context, event QuestionAttemptEvent, day time.Time, apply func(*UserStats)) (bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin stats update: %w", err)
	}
	defer tx.Rollback(ctx)

	recorded, err := recordProcessedAnswer(ctx, tx, event.AnswerID)
	if err != nil || !recorded {
		return false, err
	}
	if err := initUserStats(ctx, tx, event.UserID); err != nil {
		return false, err
	}

	if err := updateUserStats(ctx, tx, event.UserID, apply); err != nil {
		return false, err
	}
	if err := updateUserDailyStats(ctx, tx, day, event); err != nil {
		return false, err
	}
	if err := updateUserTopicStats(ctx, tx, event); err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit stats update: %w", err)
	}
	return true, nil
}

func recordProcessedAnswer(ctx context.Context, tx pgx.Tx, answerID int64) (bool, error) {
	result, err := tx.Exec(ctx, `
		INSERT INTO processed_events (event_id) VALUES ($1)
		ON CONFLICT (event_id) DO NOTHING`, answerID)
	if err != nil {
		return false, fmt.Errorf("record processed answer %d: %w", answerID, err)
	}
	return result.RowsAffected() > 0, nil
}

func initUserStats(ctx context.Context, tx pgx.Tx, userID int64) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO user_stats (user_id) VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING`, userID)
	if err != nil {
		return fmt.Errorf("initialize user stats: %w", err)
	}

	return nil
}

func updateUserStats(ctx context.Context, tx pgx.Tx, userID int64, apply func(*UserStats)) error {
	var totals UserStats
	err := tx.QueryRow(ctx, `
		SELECT xp, total_answers, correct_answers, unique_solved, active_days,
			current_streak, max_streak, current_correct_streak, best_correct_streak, last_solved_on
		FROM user_stats WHERE user_id = $1 FOR UPDATE`, userID,
	).Scan(&totals.XP, &totals.TotalAnswers, &totals.CorrectAnswers, &totals.UniqueSolved, &totals.ActiveDays,
		&totals.CurrentStreak, &totals.MaxStreak, &totals.CurrentCorrectStreak, &totals.BestCorrectStreak, &totals.LastSolvedOn)
	if err != nil {
		return fmt.Errorf("load user stats: %w", err)
	}

	apply(&totals)
	_, err = tx.Exec(ctx, `
		UPDATE user_stats SET xp = $2, total_answers = $3, correct_answers = $4, unique_solved = $5,
			active_days = $6, current_streak = $7, max_streak = $8, current_correct_streak = $9,
			best_correct_streak = $10, last_solved_on = $11
		WHERE user_id = $1`, userID, totals.XP, totals.TotalAnswers, totals.CorrectAnswers, totals.UniqueSolved,
		totals.ActiveDays, totals.CurrentStreak, totals.MaxStreak, totals.CurrentCorrectStreak, totals.BestCorrectStreak, totals.LastSolvedOn)
	if err != nil {
		return fmt.Errorf("save user stats: %w", err)
	}
	return nil
}

func updateUserDailyStats(ctx context.Context, tx pgx.Tx, day time.Time, event QuestionAttemptEvent) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO user_daily_stats (user_id, date, attempts, correct, solved, xp)
		VALUES ($1, $2, 1, $3, $4, $5)
		ON CONFLICT (user_id, date) DO UPDATE SET
			attempts = user_daily_stats.attempts + EXCLUDED.attempts,
			correct = user_daily_stats.correct + EXCLUDED.correct,
			solved = user_daily_stats.solved + EXCLUDED.solved,
			xp = user_daily_stats.xp + EXCLUDED.xp`,
		event.UserID, day, boolToInt(event.IsCorrect), boolToInt(event.FirstSolve), event.XP)
	if err != nil {
		return fmt.Errorf("update daily stats: %w", err)
	}
	return nil
}

func updateUserTopicStats(ctx context.Context, tx pgx.Tx, event QuestionAttemptEvent) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO user_topic_stats (user_id, topic, attempts, correct, solved, xp)
		VALUES ($1, $2, 1, $3, $4, $5)
		ON CONFLICT (user_id, topic) DO UPDATE SET
			attempts = user_topic_stats.attempts + EXCLUDED.attempts,
			correct = user_topic_stats.correct + EXCLUDED.correct,
			solved = user_topic_stats.solved + EXCLUDED.solved,
			xp = user_topic_stats.xp + EXCLUDED.xp`,
		event.UserID, event.Topic, boolToInt(event.IsCorrect), boolToInt(event.FirstSolve), event.XP)
	if err != nil {
		return fmt.Errorf("update topic stats: %w", err)
	}
	return nil
}

func (r *Repository) GetXP(ctx context.Context, userID int64) (int64, error) {
	var xp int64
	if err := r.db.QueryRow(ctx, "SELECT xp FROM user_stats WHERE user_id = $1", userID).Scan(&xp); err != nil {
		return 0, fmt.Errorf("get user XP: %w", err)
	}
	return xp, nil
}

func (r *Repository) GetCurrentStreak(ctx context.Context, userID int64) (int64, error) {
	var streak int64
	if err := r.db.QueryRow(ctx, "SELECT current_streak FROM user_stats WHERE user_id = $1", userID).Scan(&streak); err != nil {
		return 0, fmt.Errorf("get current streak: %w", err)
	}
	return streak, nil
}

func (r *Repository) GetCurrentCorrectStreak(ctx context.Context, userID int64) (int64, error) {
	var streak int64
	if err := r.db.QueryRow(ctx, "SELECT current_correct_streak FROM user_stats WHERE user_id = $1", userID).Scan(&streak); err != nil {
		return 0, fmt.Errorf("get current correct streak: %w", err)
	}
	return streak, nil
}

func (r *Repository) GetStats(ctx context.Context, userID int64) (UserStatsDTO, error) {
	var totals UserStatsDTO
	err := r.db.QueryRow(ctx, `
		SELECT user_id, xp, total_answers, correct_answers, unique_solved, active_days,
			current_streak, max_streak, current_correct_streak, best_correct_streak, last_solved_on
		FROM user_stats WHERE user_id = $1`, userID,
	).Scan(
		&totals.UserID, &totals.XP, &totals.TotalAnswers, &totals.CorrectAnswers, &totals.UniqueSolved, &totals.ActiveDays,
		&totals.CurrentStreak, &totals.MaxStreak, &totals.CurrentCorrectStreak, &totals.BestCorrectStreak, &totals.LastSolvedOn,
	)
	if err != nil {
		return UserStatsDTO{}, fmt.Errorf("get user stats: %w", err)
	}
	return totals, nil
}

func (r *Repository) GetDailyStats(ctx context.Context, userID int64, day time.Time) (UserDailyStats, error) {
	var stats UserDailyStats
	err := r.db.QueryRow(ctx, `
		SELECT user_id, date, attempts, correct, solved, xp, active_seconds
		FROM user_daily_stats
		WHERE user_id = $1 AND date = $2`, userID, day,
	).Scan(&stats.UserID, &stats.Date, &stats.Attempts, &stats.Correct, &stats.Solved, &stats.XP, &stats.ActiveSeconds)
	if err != nil {
		return UserDailyStats{}, fmt.Errorf("get daily stats: %w", err)
	}
	return stats, nil
}

func (r *Repository) GetDailyStatsRange(ctx context.Context, userID int64, from, through time.Time) ([]UserDailyStats, error) {
	rows, err := r.db.Query(ctx, `
		SELECT user_id, date, attempts, correct, solved, xp, active_seconds
		FROM user_daily_stats
		WHERE user_id = $1 AND date BETWEEN $2 AND $3
		ORDER BY date`, userID, from, through)
	if err != nil {
		return nil, fmt.Errorf("query daily stats range: %w", err)
	}
	stats, err := pgx.CollectRows(rows, pgx.RowToStructByPos[UserDailyStats])
	if err != nil {
		return nil, fmt.Errorf("read daily stats range: %w", err)
	}
	return stats, nil
}

func (r *Repository) GetTopicStats(ctx context.Context, userID int64, topic string) (UserTopicStats, error) {
	var stats UserTopicStats
	err := r.db.QueryRow(ctx, `
		SELECT user_id, topic, attempts, correct, solved, xp
		FROM user_topic_stats
		WHERE user_id = $1 AND topic = $2`, userID, topic,
	).Scan(&stats.UserID, &stats.Topic, &stats.Attempts, &stats.Correct, &stats.Solved, &stats.XP)
	if err != nil {
		return UserTopicStats{}, fmt.Errorf("get topic stats: %w", err)
	}
	return stats, nil
}

func (r *Repository) GetAllTopicStats(ctx context.Context, userID int64) ([]UserTopicStats, error) {
	rows, err := r.db.Query(ctx, `
		SELECT user_id, topic, attempts, correct, solved, xp
		FROM user_topic_stats
		WHERE user_id = $1
		ORDER BY topic`, userID)
	if err != nil {
		return nil, fmt.Errorf("query stats by topic: %w", err)
	}
	stats, err := pgx.CollectRows(rows, pgx.RowToStructByPos[UserTopicStats])
	if err != nil {
		return nil, fmt.Errorf("read stats by topic: %w", err)
	}
	return stats, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
