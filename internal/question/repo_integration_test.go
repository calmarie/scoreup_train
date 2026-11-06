package question

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/calmarie/scoreup_train/internal/config"
	"github.com/calmarie/scoreup_train/internal/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func TestRepository_GetQuestion_Success(t *testing.T) {
	ctx := context.Background()
	repo, db := newTestRepository(t, ctx)

	//insert test data

	const testData = `INSERT INTO questions (
		type, difficulty, topic, skill, text, options, correct_answer, explanation, data
		)
		VALUES
		(
		'radio',
		1,
		'тест топик',
		'тест скилл',
		'тест текст вопроса',
		'["1","2","3","4"]'::jsonb,
		'1'::jsonb,
		'тест обьяснение',
		'{"addition":"какие-то доп данные"}'::jsonb
		)`

	_, err := db.Exec(ctx, testData)
	if err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	//call getQuestion

	question, err := repo.GetQuestion(ctx)
	if err != nil {
		t.Fatalf("call GetQuestion: %v", err)
	}

	if question.ID != 1 {
		t.Errorf("id = %d, want %d", question.ID, 1)
	}

	if question.Type != "radio" {
		t.Errorf("type = %q, want %q", question.Type, "radio")
	}

	if question.Difficulty != 1 {
		t.Errorf("difficulty = %d, want %d", question.Difficulty, 1)
	}

	if question.Topic != "тест топик" {
		t.Errorf("topic = %q, want %q", question.Topic, "тест топик")
	}

	if question.Skill == nil {
		t.Errorf("skill = nil, want %q", "тест скилл")
	} else if *question.Skill != "тест скилл" {
		t.Errorf("skill = %q, want %q", *question.Skill, "тест скилл")
	}

	if question.Text != "тест текст вопроса" {
		t.Errorf("text = %q, want %q", question.Text, "тест текст вопроса")
	}

	if question.ImageURL != nil {
		t.Errorf("image_url = %q, want nil", *question.ImageURL)
	}

	var options []string
	err = json.Unmarshal(question.Options, &options)
	if err != nil {
		t.Fatalf("decode options: %v", err)
	}

	if !reflect.DeepEqual(options, []string{"1", "2", "3", "4"}) {
		t.Errorf("options = %q, want %q", options, []string{"1", "2", "3", "4"})
	}

	var correctAnswer int
	if err := json.Unmarshal(question.CorrectAnswer, &correctAnswer); err != nil {
		t.Fatalf("decode correct_answer: %v", err)
	}
	if correctAnswer != 1 {
		t.Errorf("correct_answer = %d, want 1", correctAnswer)
	}

	if question.Explanation == nil {
		t.Errorf("explanation = nil, want %q", "тест обьяснение")
	} else if *question.Explanation != "тест обьяснение" {
		t.Errorf("explanation = %q, want %q", *question.Explanation, "тест обьяснение")
	}

	var data map[string]string
	err = json.Unmarshal(question.Data, &data)
	if err != nil {
		t.Fatalf("decode data: %v", err)
	}

	if !reflect.DeepEqual(data, map[string]string{"addition": "какие-то доп данные"}) {
		t.Errorf("data = %v, want %v", data, map[string]string{"addition": "какие-то доп данные"})
	}

	if question.IsActive != true {
		t.Errorf("is_active = %t, want %t", question.IsActive, true)
	}
}

func TestRepository_GetQuestion_NotFound(t *testing.T) {
	ctx := context.Background()
	repo, _ := newTestRepository(t, ctx)

	//call getQuestion

	_, err := repo.GetQuestion(ctx)
	if !errors.Is(err, ErrQuestionNotFound) {
		t.Fatalf("expected ErrQuestionNotFound, got %v", err)
	}

}

func newTestRepository(t *testing.T, ctx context.Context) (*Repository, *pgxpool.Pool) {
	t.Helper()

	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatalf("load env file: %v", err)
	}

	databaseURL, err := config.DbURL("POSTGRES_DB_TEST")
	if err != nil {
		t.Fatalf("read postgres configuration: %v", err)
	}

	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to postgres: %v", err)
	}
	t.Cleanup(db.Close)

	const cleanTableQuery = `TRUNCATE questions RESTART IDENTITY CASCADE;`
	if _, err := db.Exec(ctx, cleanTableQuery); err != nil {
		t.Fatalf("clean questions table: %v", err)
	}

	return NewRepository(db), db
}

func TestGetCorrectAnswer_Success(t *testing.T) {
	ctx := context.Background()
	repo, db := newTestRepository(t, ctx)

	const testData = `INSERT INTO questions (
		type, difficulty, topic, skill, text, options, correct_answer, explanation, data
		)
		VALUES
		(
		'radio',
		1,
		'тест топик',
		'тест скилл',
		'тест текст вопроса',
		'["1","2","3","4"]'::jsonb,
		'1'::jsonb,
		'тест обьяснение',
		'{"addition":"какие-то доп данные"}'::jsonb
		)`

	_, err := db.Exec(ctx, testData)
	if err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	res, err := repo.GetCorrectAnswer(ctx, 1)

	if err != nil {
		t.Fatalf("GetCorrectAnswer: %v", err)
	}

	var got int
	if err := json.Unmarshal(res, &got); err != nil {
		t.Fatalf("decode correct answer: %v", err)
	}

	if got != 1 {
		t.Fatalf("got %v, want %v", res, 1)
	}

}

func TestSaveAnswer_Success(t *testing.T) {

	ctx := context.Background()

	repo, db := newTestRepository(t, ctx)

	var userID int64
	if err := db.QueryRow(ctx, `
		INSERT INTO users DEFAULT VALUES
		RETURNING id;
	`).Scan(&userID); err != nil {
		t.Fatalf("create user: %v", err)
	}

	var questionID int
	if err := db.QueryRow(ctx, `
		INSERT INTO questions (
			type, difficulty, topic, text, options, correct_answer, data
		)
		VALUES (
			'radio', 1, 'test topic', 'test question',
			'["1","2","3","4"]'::jsonb,
			'1'::jsonb,
			'{}'::jsonb
		)
		RETURNING id;
	`).Scan(&questionID); err != nil {
		t.Fatalf("create question: %v", err)
	}

	ans := Answer{
		User_ID:     userID,
		Question_ID: questionID,
		Answer:      json.RawMessage(`1`),
		Is_correct:  true,
	}

	res, err := repo.SaveAnswer(ctx, ans)

	if err != nil {
		t.Fatalf("SaveAnswer: %v", err)
	}

	if res.ID != 1 {
		t.Fatalf("got %v, want %v", res.ID, `1`)
	}

	if res.User_ID != userID {
		t.Fatalf("got %d, want %d", res.User_ID, userID)
	}

	if res.Question_ID != questionID {
		t.Fatalf("got %d, want %d", res.Question_ID, questionID)
	}

	if !bytes.Equal(res.Answer, json.RawMessage(`1`)) {
		t.Fatalf("got %s, want %s", res.Answer, `1`)
	}

	if !res.Is_correct {
		t.Fatalf("got %t, want %t", res.Is_correct, true)
	}

}
