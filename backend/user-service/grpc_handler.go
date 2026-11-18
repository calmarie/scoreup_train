package main

import (
	"context"
	"errors"

	pb "github.com/calmarie/scoreup_train/backend/gen/user"
	"github.com/calmarie/scoreup_train/backend/user-service/internal/users"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *userGRPCserver) CreateUser(
	ctx context.Context,
	req *pb.CreateUserRequest,
) (*pb.UserReply, error) {

	user, err := s.usersService.CreateUser(ctx, req.TelegramId)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.UserReply{
		Id:         user.ID,
		TelegramId: user.TelegramID,
		CreatedAt:  timestamppb.New(user.CreatedAt),
		XpTotal:    user.XPTotal,
	}, nil

}

func (s *userGRPCserver) GetUserByTelegramID(
	ctx context.Context,
	req *pb.GetUserByTelegramIDRequest,
) (*pb.UserReply, error) {

	user, err := s.usersService.GetUserByTelegramID(ctx, req.TelegramId)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.UserReply{
		Id:         user.ID,
		TelegramId: user.TelegramID,
		CreatedAt:  timestamppb.New(user.CreatedAt),
		XpTotal:    user.XPTotal,
	}, nil

}

func toGRPCError(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "request timed out")
	case errors.Is(err, users.ErrUserNotFound):
		return status.Error(codes.NotFound, "user not found")
	case errors.Is(err, users.ErrUserAlreadyExists):
		return status.Error(codes.AlreadyExists, "user already exists")
	case errors.Is(err, users.ErrDatabaseUnavailable):
		return status.Error(codes.Unavailable, "database unavailable")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
