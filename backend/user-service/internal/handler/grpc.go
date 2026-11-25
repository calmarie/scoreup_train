package handler

import (
	"context"
	"errors"

	pb "github.com/calmarie/scoreup_train/backend/gen/user"
	"github.com/calmarie/scoreup_train/backend/user-service/internal/users"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	pb.UnimplementedUserServer
	service *users.Service
}

func NewHandler(service *users.Service) *Handler {
	return &Handler{service: service}
}

func (s *Handler) CreateUser(
	ctx context.Context,
	req *pb.CreateUserRequest,
) (*pb.AuthReply, error) {
	if req.GetTelegramId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid telegram id")
	}

	result, err := s.service.CreateUser(ctx, req.TelegramId)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.AuthReply{
		User: &pb.UserReply{
			Id:         result.User.ID,
			TelegramId: result.User.TelegramID,
			CreatedAt:  timestamppb.New(result.User.CreatedAt),
		},
		AccessToken:  result.Session.AccessToken,
		RefreshToken: result.Session.RefreshToken,
		ExpiresIn:    result.Session.ExpiresIn,
		Created:      result.Created,
	}, nil

}

func (s *Handler) GetUserByTelegramID(
	ctx context.Context,
	req *pb.GetUserByTelegramIDRequest,
) (*pb.UserReply, error) {

	user, err := s.service.GetUserByTelegramID(ctx, req.TelegramId)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.UserReply{
		Id:         user.ID,
		TelegramId: user.TelegramID,
		CreatedAt:  timestamppb.New(user.CreatedAt),
	}, nil

}

func (s *Handler) RefreshAccess(ctx context.Context, req *pb.RefreshAccessRequest) (*pb.RefreshAccessReply, error) {
	if req.GetTelegramId() <= 0 || req.GetRefreshToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid refresh request")
	}
	session, err := s.service.RefreshAccess(ctx, req.GetRefreshToken(), req.GetTelegramId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.RefreshAccessReply{
		AccessToken:  session.AccessToken,
		RefreshToken: session.RefreshToken,
		ExpiresIn:    session.ExpiresIn,
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
