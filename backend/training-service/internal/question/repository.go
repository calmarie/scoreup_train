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

func postgresUnavailable(code string) bool {
	// Connection errors, insufficient resources, and server shutdown/startup.
	return len(code) >= 2 && (code[:2] == "08" || code[:2] == "53") ||
		code == "57P01" || code == "57P02" || code == "57P03"
}

func (r *Repository) SaveAnswer(ctx context.Context, ans Answer) (Answer, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Answer{}, fmt.Errorf("begin save answer: %w", err)
	}
	defer tx.Rollback(ctx)

	const sql = `
		INSERT INTO user_answers (user_id, question_id, answer, is_correct)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, question_id, answer, is_correct, created_at,
			(SELECT xp FROM questions WHERE id = user_answers.question_id);`

	var res Answer
	if err := tx.QueryRow(ctx, sql, ans.UserID, ans.QuestionID, ans.Answer, ans.IsCorrect).Scan(
		&res.ID,
		&res.UserID,
		&res.QuestionID,
		&res.Answer,
		&res.IsCorrect,
		&res.CreatedAt,
		&res.XP,
	); err != nil {
		return Answer{}, fmt.Errorf("insert answer: %w", err)
	}

	if res.IsCorrect {
		res.FirstSolve, err = markQuestionSolved(ctx, tx, res)
		if err != nil {
			return Answer{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Answer{}, fmt.Errorf("commit save answer: %w", err)
	}
	return res, nil
}

func markQuestionSolved(ctx context.Context, tx pgx.Tx, answer Answer) (bool, error) {
	const sql = `
		INSERT INTO user_solved_questions (user_id, question_id, solved_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, question_id) DO NOTHING;`

	tag, err := tx.Exec(ctx, sql, answer.UserID, answer.QuestionID, answer.CreatedAt)
	if err != nil {
		return false, fmt.Errorf("mark question solved: %w", err)
	}
	return tag.RowsAffected() == 1, nil
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrQuestionNotFound
		}
		return nil, fmt.Errorf("SaveAnswer db error: %w", err)
	}

	return res, nil
}
