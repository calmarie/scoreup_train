package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/calmarie/scoreup_train/backend/user-service/internal/auth/refresh"
)

type signerStub struct{}

type recordingSigner struct{ userID int64 }

func (s *recordingSigner) Sign(userID int64) (string, error) {
	s.userID = userID
	return "signed-access", nil
}

func (s signerStub) Sign(userID int64) (string, error) {
	return "signed-access", nil
}

type refreshStoreStub struct {
	hash   string
	userID int64
	err    error
}

func (s *refreshStoreStub) Get(_ context.Context, hash string) (int64, error) {
	s.hash = hash
	return s.userID, s.err
}

func TestRefreshAccessUsesStoredUserID(t *testing.T) {
	store := &refreshStoreStub{userID: 42}
	signer := &recordingSigner{}
	session, err := NewIssuer(signer, store).RefreshAccess(context.Background(), "refresh-token")
	if err != nil || session.AccessToken != "signed-access" || session.RefreshToken != "refresh-token" || signer.userID != 42 {
		t.Fatalf("session = %+v, error = %v", session, err)
	}
	if store.hash != refresh.HashRefreshToken("refresh-token") {
		t.Fatal("refresh token was not hashed before lookup")
	}
}

func TestRefreshAccessRejectsExpiredToken(t *testing.T) {
	signer := &recordingSigner{}
	store := &refreshStoreStub{err: refresh.ErrRefreshTokenInvalid}
	session, err := NewIssuer(signer, store).RefreshAccess(context.Background(), "expired")
	if !errors.Is(err, refresh.ErrRefreshTokenInvalid) || session.AccessToken != "" || signer.userID != 0 {
		t.Fatalf("session = %+v, error = %v, signed for %d", session, err, signer.userID)
	}
}

func TestRefreshAccessRejectsStoreFailure(t *testing.T) {
	storeErr := errors.New("redis unavailable")
	store := &refreshStoreStub{err: storeErr}
	session, err := NewIssuer(signerStub{}, store).RefreshAccess(context.Background(), "refresh-token")
	if !errors.Is(err, storeErr) || session.AccessToken != "" {
		t.Fatalf("session = %+v, error = %v", session, err)
	}
}

func (s *refreshStoreStub) Save(_ context.Context, hash string, userID int64) error {
	s.hash, s.userID = hash, userID
	return s.err
}

func TestIssueSessionStoresOnlyRefreshHash(t *testing.T) {
	store := &refreshStoreStub{}
	session, err := NewIssuer(signerStub{}, store).IssueSession(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if session.AccessToken != "signed-access" || session.RefreshToken == "" || session.ExpiresIn != 900 {
		t.Fatalf("session = %+v", session)
	}
	if store.userID != 42 || store.hash != refresh.HashRefreshToken(session.RefreshToken) || store.hash == session.RefreshToken {
		t.Fatal("refresh token was not stored as a hash for user 42")
	}
}

func TestIssueSessionDoesNotReturnTokensWhenStoreFails(t *testing.T) {
	store := &refreshStoreStub{err: errors.New("redis unavailable")}
	session, err := NewIssuer(signerStub{}, store).IssueSession(context.Background(), 42)
	if err == nil || session.AccessToken != "" || session.RefreshToken != "" {
		t.Fatalf("session = %+v, error = %v", session, err)
	}
}
