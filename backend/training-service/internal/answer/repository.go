package answer

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/calmarie/scoreup_train/backend/training-service/internal/outbox"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnswerRepository interface {
	SaveAnswer(ctx context.Context, ans Answer) (Answer, error)
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// SaveAnswer stores the answer and its outbox event in one transaction.
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
			(SELECT xp FROM questions WHERE id = user_answers.question_id),
			(SELECT topic FROM questions WHERE id = user_answers.question_id),
			(SELECT difficulty FROM questions WHERE id = user_answers.question_id);`

	var res Answer
	if err := tx.QueryRow(ctx, sql, ans.UserID, ans.QuestionID, ans.Answer, ans.IsCorrect).Scan(
		&res.ID,
		&res.UserID,
		&res.QuestionID,
		&res.Answer,
		&res.IsCorrect,
		&res.CreatedAt,
		&res.XP,
		&res.Topic,
		&res.Difficulty,
	); err != nil {
		return Answer{}, fmt.Errorf("insert answer: %w", err)
	}

	if res.IsCorrect {
		res.FirstSolve, err = markQuestionSolved(ctx, tx, res)
		if err != nil {
			return Answer{}, err
		}
	}

	payload, err := json.Marshal(res.AttemptEvent())

	if err != nil {
		return Answer{}, fmt.Errorf("encode attempt event: %w", err)
	}

	if _, err := outbox.SaveEvent(ctx, tx, outbox.Event{
		Topic:    AttemptTopic,
		EventKey: strconv.FormatInt(res.UserID, 10),
		Payload:  payload,
	}); err != nil {
		return Answer{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Answer{}, fmt.Errorf("commit save answer: %w", err)
	}
	return res, nil
}

// markQuestionSolved reports whether this transaction recorded the user's first solve for the question.
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
