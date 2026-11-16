package solvedquestions

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) MarkQuestionSolved(ctx context.Context, userID int64, questionID int) (bool, error) {
	return markQuestionSolved(ctx, r.db, userID, questionID)
}

type sqlExecutor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func markQuestionSolved(ctx context.Context, db sqlExecutor, userID int64, questionID int) (bool, error) {
	const query = `
		INSERT INTO user_solved_questions (user_id, question_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, question_id) DO NOTHING`

	tag, err := db.Exec(ctx, query, userID, questionID)
	if err != nil {
		return false, fmt.Errorf("mark question solved: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *Repository) AwardQuestionXP(ctx context.Context, userID int64, questionID int, xp int) (bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin XP award: %w", err)
	}
	defer tx.Rollback(ctx)

	marked, err := markQuestionSolved(ctx, tx, userID, questionID)
	if err != nil {
		return false, err
	}
	if marked {
		if _, err := tx.Exec(ctx, `UPDATE users SET xp_total = xp_total + $1 WHERE id = $2`, xp, userID); err != nil {
			return false, fmt.Errorf("add question XP: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit XP award: %w", err)
	}
	return marked, nil
}
