package answer

import (
	"bytes"
	"context"
	"encoding/json"
)

type correctAnswerReader interface {
	GetCorrectAnswer(ctx context.Context, questionID int) (json.RawMessage, error)
}

type Service struct {
	repository AnswerRepository
	questions  correctAnswerReader
}

func NewService(repository AnswerRepository, questions correctAnswerReader) *Service {
	return &Service{repository: repository, questions: questions}
}

func (s *Service) SaveAnswer(ctx context.Context, userAns SubmitAnswerRequest) (Answer, error) {
	correctAns, err := s.questions.GetCorrectAnswer(ctx, userAns.QuestionID)

	if err != nil {
		return Answer{}, err
	}

	// (hardcoded) Byte equality makes JSON formatting and array order part of answer matching.
	isCorrect := bytes.Equal(correctAns, userAns.Answer)

	res := Answer{
		UserID:     userAns.UserID,
		QuestionID: userAns.QuestionID,
		Answer:     userAns.Answer,
		IsCorrect:  isCorrect,
	}

	return s.repository.SaveAnswer(ctx, res)
}
