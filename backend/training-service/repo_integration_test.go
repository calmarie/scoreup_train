package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/calmarie/scoreup_train/backend/training-service/internal/answer"
	"github.com/calmarie/scoreup_train/backend/training-service/internal/question"
	"github.com/calmarie/scoreup_train/backend/training-service/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func TestRepository_GetQuestion_Success(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t, ctx)
	repo := question.NewRepository(db)

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
	if question.XP != 100 {
		t.Errorf("xp = %d, want 100", question.XP)
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
	repo := question.NewRepository(newTestDB(t, ctx))

	//call getQuestion

	_, err := repo.GetQuestion(ctx)
	if !errors.Is(err, question.ErrQuestionNotFound) {
		t.Fatalf("expected ErrQuestionNotFound, got %v", err)
	}

}

func newTestDB(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatalf("load env file: %v", err)
	}
	testUser := os.Getenv("TRAINING_TEST_POSTGRES_USER")
	if testUser == "" {
		testUser = "postgres"
	}
	t.Setenv("POSTGRES_USER", testUser)
	testPassword := os.Getenv("TRAINING_TEST_POSTGRES_PASSWORD")
	if testPassword == "" {
		testPassword = "postgres"
	}
	t.Setenv("POSTGRES_PASSWORD", testPassword)
	testPort := os.Getenv("TRAINING_TEST_POSTGRES_PORT")
	if testPort == "" {
		testPort = "5434"
	}
	t.Setenv("POSTGRES_PORT", testPort)
	t.Setenv("POSTGRES_DB", os.Getenv("POSTGRES_DB_TEST"))

	databaseURL, err := postgres.DatabaseURL()
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

	return db
}

func TestGetCorrectAnswer_Success(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t, ctx)
	repo := question.NewRepository(db)

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

func TestGetCorrectAnswer_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := question.NewRepository(newTestDB(t, ctx))

	_, err := repo.GetCorrectAnswer(ctx, 999999)
	if !errors.Is(err, question.ErrQuestionNotFound) {
		t.Fatalf("error = %v, want ErrQuestionNotFound", err)
	}
}

func TestSaveAnswer_Success(t *testing.T) {

	ctx := context.Background()

	db := newTestDB(t, ctx)
	repo := answer.NewRepository(db)

	const userID int64 = 42

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

	ans := answer.Answer{
		UserID:     userID,
		QuestionID: questionID,
		Answer:     json.RawMessage(`1`),
		IsCorrect:  true,
	}

	res, err := repo.SaveAnswer(ctx, ans)

	if err != nil {
		t.Fatalf("SaveAnswer: %v", err)
	}

	if res.ID != 1 {
		t.Fatalf("got %v, want %v", res.ID, `1`)
	}
	if res.XP != 100 {
		t.Fatalf("xp = %d, want 100", res.XP)
	}
	if res.Topic != "test topic" || res.Difficulty != 1 {
		t.Fatalf("answer metadata = %q, %d, want test topic, 1", res.Topic, res.Difficulty)
	}

	if res.UserID != userID {
		t.Fatalf("got %d, want %d", res.UserID, userID)
	}

	if res.QuestionID != questionID {
		t.Fatalf("got %d, want %d", res.QuestionID, questionID)
	}

	if !bytes.Equal(res.Answer, json.RawMessage(`1`)) {
		t.Fatalf("got %s, want %s", res.Answer, `1`)
	}

	if !res.IsCorrect {
		t.Fatalf("got %t, want %t", res.IsCorrect, true)
	}

	if !res.FirstSolve {
		t.Fatal("first correct answer was not marked as the first solve")
	}

	repeat, err := repo.SaveAnswer(ctx, ans)
	if err != nil || repeat.FirstSolve || repeat.ID == res.ID {
		t.Fatalf("save repeated answer = %+v, %v", repeat, err)
	}

	ans.UserID++
	ans.IsCorrect = false
	wrong, err := repo.SaveAnswer(ctx, ans)
	if err != nil || wrong.FirstSolve {
		t.Fatalf("save incorrect answer = %+v, %v", wrong, err)
	}
	ans.IsCorrect = true
	correct, err := repo.SaveAnswer(ctx, ans)
	if err != nil || !correct.FirstSolve {
		t.Fatalf("save first correct answer after an incorrect attempt = %+v, %v", correct, err)
	}
}
