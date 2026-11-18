package stats

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type StatsRepository interface {
	Update(ctx context.Context, userID, answerID int64, apply func(*UserStats)) (bool, error)
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Update applies each answer once, holding the user's row lock until the new totals are committed.
func (r *Repository) Update(ctx context.Context, userID, answerID int64, apply func(*UserStats)) (bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin stats update: %w", err)
	}
	defer tx.Rollback(ctx)

	result, err := tx.Exec(ctx, `
		INSERT INTO processed_events (event_id) VALUES ($1)
		ON CONFLICT (event_id) DO NOTHING`, answerID)
	if err != nil {
		return false, fmt.Errorf("record processed answer %d: %w", answerID, err)
	}
	if result.RowsAffected() == 0 {
		return false, nil
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO user_stats (user_id) VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING`, userID)
	if err != nil {
		return false, fmt.Errorf("initialize user stats: %w", err)
	}

	var totals UserStats
	err = tx.QueryRow(ctx, `
		SELECT xp, total_answers, correct_answers, unique_solved, active_days,
			current_streak, max_streak, current_correct_streak, best_correct_streak, last_solved_on
		FROM user_stats WHERE user_id = $1 FOR UPDATE`, userID,
	).Scan(&totals.XP, &totals.TotalAnswers, &totals.CorrectAnswers, &totals.UniqueSolved, &totals.ActiveDays,
		&totals.CurrentStreak, &totals.MaxStreak, &totals.CurrentCorrectStreak, &totals.BestCorrectStreak, &totals.LastSolvedOn)
	if err != nil {
		return false, fmt.Errorf("load user stats: %w", err)
	}

	apply(&totals)
	_, err = tx.Exec(ctx, `
		UPDATE user_stats SET xp = $2, total_answers = $3, correct_answers = $4, unique_solved = $5,
			active_days = $6, current_streak = $7, max_streak = $8, current_correct_streak = $9,
			best_correct_streak = $10, last_solved_on = $11
		WHERE user_id = $1`, userID, totals.XP, totals.TotalAnswers, totals.CorrectAnswers, totals.UniqueSolved,
		totals.ActiveDays, totals.CurrentStreak, totals.MaxStreak, totals.CurrentCorrectStreak, totals.BestCorrectStreak, totals.LastSolvedOn)
	if err != nil {
		return false, fmt.Errorf("save user stats: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit stats update: %w", err)
	}
	return true, nil
}
