package users

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/calmarie/scoreup_train/backend/user-service/internal/auth"
	"github.com/calmarie/scoreup_train/backend/user-service/internal/auth/refresh"
)

type createUserRepository struct {
	user    User
	err     error
	getUser User
}

func (r createUserRepository) CreateUser(context.Context, int64) (User, error) { return r.user, r.err }
func (r createUserRepository) GetUserByTelegramID(context.Context, int64) (User, error) {
	return r.getUser, nil
}

type sessionIssuerStub struct {
	gotUserID int64
	session   auth.Session
	err       error
}

func (i *sessionIssuerStub) IssueSession(_ context.Context, userID int64) (auth.Session, error) {
	i.gotUserID = userID
	return i.session, i.err
}

func (i *sessionIssuerStub) RefreshAccess(_ context.Context, token string) (auth.Session, error) {
	if token != "refresh" {
		return auth.Session{}, errors.New("unexpected refresh token")
	}
	return i.session, i.err
}

func TestRefreshAccessDelegatesToIssuer(t *testing.T) {
	issuer := &sessionIssuerStub{session: auth.Session{AccessToken: "new-access"}}
	got, err := NewService(createUserRepository{}, issuer).RefreshAccess(context.Background(), "refresh", 123)
	if err != nil || got.AccessToken != "new-access" {
		t.Fatalf("session = %+v, error = %v", got, err)
	}
}

type expiredSessionIssuer struct{ gotUserID int64 }

func (i *expiredSessionIssuer) RefreshAccess(context.Context, string) (auth.Session, error) {
	return auth.Session{}, refresh.ErrRefreshTokenInvalid
}

func (i *expiredSessionIssuer) IssueSession(_ context.Context, userID int64) (auth.Session, error) {
	i.gotUserID = userID
	return auth.Session{AccessToken: "new-access", RefreshToken: "new-refresh"}, nil
}

func TestRefreshAccessIssuesSessionForExpiredToken(t *testing.T) {
	issuer := &expiredSessionIssuer{}
	repo := createUserRepository{getUser: User{ID: 42, TelegramID: 123}}
	session, err := NewService(repo, issuer).RefreshAccess(context.Background(), "expired", 123)
	if err != nil || issuer.gotUserID != 42 || session.AccessToken != "new-access" || session.RefreshToken != "new-refresh" {
		t.Fatalf("session = %+v, userID = %d, error = %v", session, issuer.gotUserID, err)
	}
}

func TestRefreshAccessDoesNotIssueSessionOnRedisError(t *testing.T) {
	storeErr := errors.New("redis unavailable")
	issuer := &sessionIssuerStub{err: storeErr}
	_, err := NewService(createUserRepository{getUser: User{ID: 42}}, issuer).RefreshAccess(context.Background(), "refresh", 123)
	if !errors.Is(err, storeErr) || issuer.gotUserID != 0 {
		t.Fatalf("error = %v, issued for %d", err, issuer.gotUserID)
	}
}

func TestCreateUserReturnsUserAndSession(t *testing.T) {
	wantUser := User{ID: 42, TelegramID: 123, CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}
	issuer := &sessionIssuerStub{session: auth.Session{AccessToken: "access", RefreshToken: "refresh"}}
	service := NewService(createUserRepository{user: wantUser}, issuer)

	got, err := service.CreateUser(context.Background(), 123)
	if err != nil {
		t.Fatal(err)
	}
	if got.User != wantUser || got.Session.AccessToken != "access" || got.Session.RefreshToken != "refresh" || !got.Created {
		t.Fatalf("result = %+v", got)
	}
	if issuer.gotUserID != 42 {
		t.Fatalf("issued for user %d, want 42", issuer.gotUserID)
	}
}

func TestCreateUserDoesNotIssueSessionWhenInsertFails(t *testing.T) {
	issuer := &sessionIssuerStub{}
	service := NewService(createUserRepository{err: ErrDatabaseUnavailable}, issuer)
	_, err := service.CreateUser(context.Background(), 123)
	if !errors.Is(err, ErrDatabaseUnavailable) || issuer.gotUserID != 0 {
		t.Fatalf("error = %v; issued for user %d", err, issuer.gotUserID)
	}
}

func TestCreateUserIssuesSessionForExistingUser(t *testing.T) {
	issuer := &sessionIssuerStub{session: auth.Session{AccessToken: "access", RefreshToken: "refresh"}}
	repo := createUserRepository{err: ErrUserAlreadyExists, getUser: User{ID: 42, TelegramID: 123}}
	result, err := NewService(repo, issuer).CreateUser(context.Background(), 123)
	if err != nil {
		t.Fatal(err)
	}
	if result.Created || result.User.ID != 42 || issuer.gotUserID != 42 || result.Session.RefreshToken != "refresh" {
		t.Fatalf("result = %+v, issuer user ID = %d", result, issuer.gotUserID)
	}
}

func TestCreateUserWithoutIssuerDoesNotInsert(t *testing.T) {
	service := NewService(createUserRepository{user: User{ID: 42}}, nil)
	_, err := service.CreateUser(context.Background(), 123)
	if err == nil {
		t.Fatal("expected error when session issuer is unavailable")
	}
}
