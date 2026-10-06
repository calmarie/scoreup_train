package question

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type QuestionRepository interface {
	GetQuestion(ctx context.Context) (Question, error)
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetQuestion(ctx context.Context) (Question, error) {
	const questionQuery = `
		SELECT
			id,
			type,
			difficulty,
			topic,
			skill,
			text,
			image_url,
			options,
			correct_answer,
			explanation,
			data,
			is_active
		FROM questions
		WHERE is_active = TRUE
		ORDER BY RANDOM()
		LIMIT 1;`

	var question Question
	if err := r.db.QueryRow(ctx, questionQuery).Scan(
		&question.ID,
		&question.Type,
		&question.Difficulty,
		&question.Topic,
		&question.Skill,
		&question.Text,
		&question.ImageURL,
		&question.Options,
		&question.CorrectAnswer,
		&question.Explanation,
		&question.Data,
		&question.IsActive,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Question{}, ErrQuestionNotFound
		}

		if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			var networkErr net.Error
			var postgresErr *pgconn.PgError
			if errors.As(err, &networkErr) || (errors.As(err, &postgresErr) && postgresUnavailable(postgresErr.Code)) {
				return Question{}, fmt.Errorf("select random question: %w: %w", ErrDatabaseUnavailable, err)
			}
		}
		return Question{}, fmt.Errorf("select random question: %w", err)
	}

	return question, nil
}

func postgresUnavailable(code string) bool {
	// Connection errors, insufficient resources, and server shutdown/startup.
	return len(code) >= 2 && (code[:2] == "08" || code[:2] == "53") ||
		code == "57P01" || code == "57P02" || code == "57P03"
}
