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
	SaveAnswer(ctx context.Context, ans Answer) (Answer, error)
	GetCorrectAnswer(ctx context.Context, question_id int) (json.RawMessage, error)
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

func (r *Repository) SaveAnswer(ctx context.Context, ans Answer) (Answer, error) {

	sql := `INSERT INTO user_answers (user_id, question_id, answer, is_correct) 
	VALUES($1, $2, $3, $4) 
	RETURNING id, user_id, question_id, answer, is_correct, created_at;`

	row := r.db.QueryRow(ctx, sql, ans.User_ID, ans.Question_ID, ans.Answer, ans.Is_correct)

	var res Answer
	err := row.Scan(
		&res.ID,
		&res.User_ID,
		&res.Question_ID,
		&res.Answer,
		&res.Is_correct,
		&res.Created_at,
	)

	if err != nil {
		return Answer{}, fmt.Errorf("SaveAnswer db error: %w", err)
	}

	return res, nil
}

func (r *Repository) GetCorrectAnswer(ctx context.Context, question_id int) (json.RawMessage, error) {

	sql := `SELECT correct_answer FROM questions
	WHERE id = $1; `

	row := r.db.QueryRow(ctx, sql, question_id)

	var res json.RawMessage
	err := row.Scan(
		&res,
	)

	if err != nil {
		return nil, fmt.Errorf("SaveAnswer db error: %w", err)
	}

	return res, nil
}
