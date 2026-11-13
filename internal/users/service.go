package users

import "context"

type UserService interface {
	CreateUser(ctx context.Context) (User, error)
	GetUserByTelegramID(ctx context.Context, telegramID int64) (User, error)
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

func (s *Service) GetUserByTelegramID(ctx context.Context, telegramID int64) (User, error) {
	return s.repository.GetUserByTelegramID(ctx, telegramID)
}
