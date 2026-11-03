package users

import "context"

type UserService interface {
	CreateUser(ctx context.Context) (User, error)
}

type Service struct {
	repository UserRepository
}

func NewService(repository UserRepository) *Service {
	return &Service{repository: repository}
}

func (s *Service) CreateUser(ctx context.Context) (User, error) {
	return s.repository.CreateUser(ctx)
}
