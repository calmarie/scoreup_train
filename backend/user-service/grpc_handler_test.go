package main

import (
	"testing"

	"github.com/calmarie/scoreup_train/backend/user-service/internal/users"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestDuplicateUserMapsToAlreadyExists(t *testing.T) {
	err := toGRPCError(users.ErrUserAlreadyExists)
	if got := status.Code(err); got != codes.AlreadyExists {
		t.Errorf("gRPC code = %v, want AlreadyExists", got)
	}
}
