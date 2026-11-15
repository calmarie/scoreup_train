package question

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type answerRepository struct {
	saved Answer
}

func (*answerRepository) GetQuestion(context.Context) (Question, error) {
	return Question{}, errors.New("unexpected GetQuestion call")
}

func (*answerRepository) GetCorrectAnswer(context.Context, int) (json.RawMessage, error) {
	return json.RawMessage(`"yes"`), nil
}

func (r *answerRepository) SaveAnswer(_ context.Context, answer Answer) (Answer, error) {
	r.saved = answer
	return answer, nil
}

func TestServiceSaveAnswerUsesUserIDFromRequest(t *testing.T) {
	repo := &answerRepository{}
	service := NewService(repo)
	request := SubmitAnswerRequest{
		UserID:     42,
		QuestionID: 5,
		Answer:     json.RawMessage(`"yes"`),
	}

	_, err := service.SaveAnswer(context.Background(), request)
	if err != nil {
		t.Fatalf("SaveAnswer: %v", err)
	}
	if repo.saved.UserID != 42 {
		t.Fatalf("saved user ID = %d, want 42", repo.saved.UserID)
	}
}
