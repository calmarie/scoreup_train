package handler

import (
	"context"
	"testing"
	"time"

	pb "github.com/calmarie/scoreup_train/backend/gen/user"
	"github.com/calmarie/scoreup_train/backend/user-service/internal/auth"
	"github.com/calmarie/scoreup_train/backend/user-service/internal/users"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcUserRepository struct{}

func (grpcUserRepository) CreateUser(context.Context, int64) (users.User, error) {
	return users.User{ID: 42, TelegramID: 123, CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}, nil
}
func (grpcUserRepository) GetUserByTelegramID(context.Context, int64) (users.User, error) {
	return users.User{}, nil
}

type grpcSessionIssuer struct{}

func (grpcSessionIssuer) IssueSession(context.Context, int64) (auth.Session, error) {
	return auth.Session{AccessToken: "access", RefreshToken: "refresh", ExpiresIn: 900}, nil
}

func (grpcSessionIssuer) RefreshAccess(context.Context, string) (auth.Session, error) {
	return auth.Session{}, nil
}

func TestCreateUserReplyContainsUserAndTokens(t *testing.T) {
	srv := NewHandler(users.NewService(grpcUserRepository{}, grpcSessionIssuer{}))
	reply, err := srv.CreateUser(context.Background(), &pb.CreateUserRequest{TelegramId: 123})
	if err != nil {
		t.Fatal(err)
	}
	if reply.User == nil || reply.User.Id != 42 || reply.User.TelegramId != 123 || reply.AccessToken != "access" || reply.RefreshToken != "refresh" || reply.ExpiresIn != 900 || !reply.Created {
		t.Fatalf("reply = %+v", reply)
	}
}

func TestDuplicateUserMapsToAlreadyExists(t *testing.T) {
	err := toGRPCError(users.ErrUserAlreadyExists)
	if got := status.Code(err); got != codes.AlreadyExists {
		t.Errorf("gRPC code = %v, want AlreadyExists", got)
	}
}
