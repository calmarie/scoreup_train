package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/calmarie/scoreup_train/backend/user-service/internal/auth"
	"github.com/calmarie/scoreup_train/backend/user-service/internal/auth/refresh"
)

type SessionIssuer interface {
	IssueSession(ctx context.Context, userID int64) (auth.Session, error)
	RefreshAccess(ctx context.Context, token string) (auth.Session, error)
}

type UserService interface {
	CreateUser(ctx context.Context, telegramID int64) (CreateUserResult, error)
	GetUserByTelegramID(ctx context.Context, telegramID int64) (User, error)
	RefreshAccess(ctx context.Context, token string, telegramID int64) (auth.Session, error)
}

// RefreshAccess temporarily trusts telegramID when the refresh token has expired.
// Verify the caller's Telegram identity before using this in production.
func (s *Service) RefreshAccess(ctx context.Context, token string, telegramID int64) (auth.Session, error) {
	if s.issuer == nil {
		return auth.Session{}, fmt.Errorf("refresh access: session issuer unavailable")
	}
	session, err := s.issuer.RefreshAccess(ctx, token)
	if !errors.Is(err, refresh.ErrRefreshTokenInvalid) {
		return session, err
	}
	user, err := s.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return auth.Session{}, fmt.Errorf("refresh access: find user: %w", err)
	}
	return s.issuer.IssueSession(ctx, user.ID)
}

type Service struct {
	repository UserRepository
	issuer     SessionIssuer
}

func NewService(repository UserRepository, issuer SessionIssuer) *Service {
	return &Service{repository: repository, issuer: issuer}
}

// Verify the caller's Telegram identity before using this in production.
func (s *Service) CreateUser(ctx context.Context, telegramID int64) (CreateUserResult, error) {
	if s.issuer == nil {
		return CreateUserResult{}, fmt.Errorf("create user: session issuer unavailable")
	}
	user, err := s.repository.CreateUser(ctx, telegramID)
	created := err == nil
	if errors.Is(err, ErrUserAlreadyExists) {
		user, err = s.repository.GetUserByTelegramID(ctx, telegramID)
	}
	if err != nil {
		return CreateUserResult{}, err
	}
	session, err := s.issuer.IssueSession(ctx, user.ID)
	if err != nil {
		return CreateUserResult{}, err
	}
	return CreateUserResult{User: user, Session: session, Created: created}, nil
}

func (s *Service) GetUserByTelegramID(ctx context.Context, telegramID int64) (User, error) {
	return s.repository.GetUserByTelegramID(ctx, telegramID)
}
