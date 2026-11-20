package auth

import (
	"context"
	"fmt"

	"github.com/calmarie/scoreup_train/backend/user-service/internal/auth/access"
	"github.com/calmarie/scoreup_train/backend/user-service/internal/auth/refresh"
)

type AccessSigner interface {
	Sign(userID int64) (string, error)
}

type RefreshStore interface {
	Save(ctx context.Context, hash string, userID int64) error
	Get(ctx context.Context, hash string) (int64, error)
}

// RefreshAccess signs a new access token for the owner of a valid refresh token.
func (i *Issuer) RefreshAccess(ctx context.Context, token string) (Session, error) {
	if token == "" {
		return Session{}, refresh.ErrRefreshTokenInvalid
	}
	userID, err := i.store.Get(ctx, refresh.HashRefreshToken(token))
	if err != nil {
		return Session{}, fmt.Errorf("look up refresh token: %w", err)
	}
	if userID <= 0 {
		return Session{}, fmt.Errorf("refresh access: invalid user ID")
	}
	accessToken, err := i.signer.Sign(userID)
	if err != nil {
		return Session{}, fmt.Errorf("issue access token: %w", err)
	}
	return Session{AccessToken: accessToken, RefreshToken: token, ExpiresIn: int64(access.TokenLifetime.Seconds())}, nil
}

type Session struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

type Issuer struct {
	signer AccessSigner
	store  RefreshStore
}

func NewIssuer(signer AccessSigner, store RefreshStore) *Issuer {
	return &Issuer{signer: signer, store: store}
}

func (i *Issuer) IssueSession(ctx context.Context, userID int64) (Session, error) {
	if userID <= 0 {
		return Session{}, fmt.Errorf("issue session: invalid user ID")
	}
	refreshToken, err := refresh.GenRefreshToken()
	if err != nil {
		return Session{}, fmt.Errorf("issue refresh token: %w", err)
	}
	accessToken, err := i.signer.Sign(userID)
	if err != nil {
		return Session{}, fmt.Errorf("issue access token: %w", err)
	}
	if err := i.store.Save(ctx, refresh.HashRefreshToken(refreshToken), userID); err != nil {
		return Session{}, fmt.Errorf("store refresh token: %w", err)
	}
	return Session{AccessToken: accessToken, RefreshToken: refreshToken, ExpiresIn: int64(access.TokenLifetime.Seconds())}, nil
}
