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
	answer.ID = 19
	r.saved = answer
	return answer, nil
}

func TestServiceSaveAnswerPublishesSavedAttempt(t *testing.T) {
	repo := &answerRepository{}
	var published Answer
	service := NewService(repo, func(_ context.Context, answer Answer) error {
		published = answer
		return nil
	})
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
	if published.ID != 19 || published.UserID != 42 || published.QuestionID != 5 || !published.IsCorrect {
		t.Fatalf("published attempt = %+v, want saved answer 19 for user 42 and question 5", published)
	}
}
