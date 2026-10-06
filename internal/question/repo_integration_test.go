package question

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/calmarie/scoreup_train/internal/config"
	"github.com/calmarie/scoreup_train/internal/postgres"
	"github.com/joho/godotenv"
)

func TestRepository_GetQuestion_Success(t *testing.T) {
	ctx := context.Background()

	err := godotenv.Load("../../.env")
	if err != nil {
		t.Fatalf("load env file: %v", err)
	}

	//test db connection
	databaseURL, err := config.DbURL("POSTGRES_DB_TEST")
	if err != nil {
		t.Fatalf("read postgres configuration: %v", err)
	}

	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to postgres: %v", err)
	}
	defer db.Close()

	//clean table questions
	const cleanTableQuery = `TRUNCATE questions RESTART IDENTITY CASCADE;`

	_, err = db.Exec(ctx, cleanTableQuery)
	if err != nil {
		t.Fatalf("clean questions table: %v", err)
	}

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
		'1',
		'тест обьяснение',
		'{"addition":"какие-то доп данные"}'::jsonb
		)`

	_, err = db.Exec(ctx, testData)
	if err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	//test repo

	repo := NewRepository(db)

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

	if question.CorrectAnswer != "1" {
		t.Errorf("correct_answer = %q, want %q", question.CorrectAnswer, "1")
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

	err := godotenv.Load("../../.env")
	if err != nil {
		t.Fatalf("load env file: %v", err)
	}

	//test db connection
	databaseURL, err := config.DbURL("POSTGRES_DB_TEST")
	if err != nil {
		t.Fatalf("read postgres configuration: %v", err)
	}

	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to postgres: %v", err)
	}
	defer db.Close()

	//clean table questions
	const cleanTableQuery = `TRUNCATE questions RESTART IDENTITY CASCADE;`

	_, err = db.Exec(ctx, cleanTableQuery)
	if err != nil {
		t.Fatalf("clean questions table: %v", err)
	}

	//test repo

	repo := NewRepository(db)

	//call getQuestion

	_, err = repo.GetQuestion(ctx)
	if !errors.Is(err, ErrQuestionNotFound) {
		t.Fatalf("expected ErrQuestionNotFound, got %v", err)
	}

}
