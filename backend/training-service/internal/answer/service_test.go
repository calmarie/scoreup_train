package answer

import (
	"context"
	"encoding/json"
	"testing"
)

type answerRepository struct {
	saved Answer
}

func (*answerRepository) GetCorrectAnswer(context.Context, int) (json.RawMessage, error) {
	return json.RawMessage(`"yes"`), nil
}

func (r *answerRepository) SaveAnswer(_ context.Context, answer Answer) (Answer, error) {
	answer.ID = 19
	r.saved = answer
	return answer, nil
}

func TestServiceSaveAnswer(t *testing.T) {
	repo := &answerRepository{}
	service := NewService(repo, repo)
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
