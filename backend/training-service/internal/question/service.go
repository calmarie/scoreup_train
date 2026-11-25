package question

import "context"

type Service struct {
	repository QuestionRepository
}

func NewService(repository QuestionRepository) *Service {
	return &Service{repository: repository}
}

func (s *Service) GetQuestion(ctx context.Context) (Question, error) {
	return s.repository.GetQuestion(ctx)
}
