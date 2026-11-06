package question

import (
	"bytes"
	"context"
)

type QuestistionService interface {
	GetQuestion(ctx context.Context) (Question, error)
	SaveAnswer(ctx context.Context, userAns SubmitAnswerRequest, userId int64) (Answer, error)
}

type Service struct {
	repository QuestionRepository
}

func NewService(repository QuestionRepository) *Service {
	return &Service{repository: repository}
}

func (s *Service) GetQuestion(ctx context.Context) (Question, error) {
	return s.repository.GetQuestion(ctx)
}

func (s *Service) SaveAnswer(ctx context.Context, userAns SubmitAnswerRequest, userId int64) (Answer, error) {

	correctAns, err := s.repository.GetCorrectAnswer(ctx, userAns.QuestionID)

	if err != nil {
		return Answer{}, err
	}

	isCorrect := bytes.Equal(correctAns, userAns.Answer)

	res := Answer{
		User_ID:     userId,
		Question_ID: userAns.QuestionID,
		Answer:      userAns.Answer,
		Is_correct:  isCorrect,
	}

	return s.repository.SaveAnswer(ctx, res)
}
