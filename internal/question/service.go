package question

import (
	"context"
)

type QuestistionService interface {
	GetQuestion(ctx context.Context) (Question, error)
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
