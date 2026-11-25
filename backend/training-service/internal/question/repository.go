package question

import (
	"context"
	"encoding/json"
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

// GetQuestion picks a random active question. Previously solved questions are eligible.
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
			explanation,
			data,
			is_active,
			xp
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
		&question.Explanation,
		&question.Data,
		&question.IsActive,
		&question.XP,
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

func (r *Repository) GetCorrectAnswer(ctx context.Context, questionID int) (json.RawMessage, error) {
	const query = `SELECT correct_answer FROM questions WHERE id = $1`

	var correctAnswer json.RawMessage
	if err := r.db.QueryRow(ctx, query, questionID).Scan(&correctAnswer); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrQuestionNotFound
		}
		return nil, fmt.Errorf("get correct answer: %w", err)
	}
	return correctAnswer, nil
}

func postgresUnavailable(code string) bool {
	// Connection errors, insufficient resources, and server shutdown/startup.
	return len(code) >= 2 && (code[:2] == "08" || code[:2] == "53") ||
		code == "57P01" || code == "57P02" || code == "57P03"
}
