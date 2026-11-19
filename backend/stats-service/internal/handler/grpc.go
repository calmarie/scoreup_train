package handler

import (
	"context"
	"errors"

	pb "github.com/calmarie/scoreup_train/backend/gen/stats"
	"github.com/calmarie/scoreup_train/backend/stats-service/internal/stats"
	"github.com/jackc/pgx/v5"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	pb.UnimplementedStatsServer
	service *stats.Service
}

func NewHandler(service *stats.Service) *Handler {
	return &Handler{service: service}
}

func (s *Handler) GetStats(
	ctx context.Context,
	req *pb.UserStatsRequest,
) (*pb.GetStatsReply, error) {

	totals, err := s.service.GetStats(ctx, req.GetUserId())
	if err != nil {
		return nil, toGRPCError(err)
	}

	var lastSolved *date.Date
	if totals.LastSolvedOn != nil {
		year, month, day := totals.LastSolvedOn.Date()
		lastSolved = &date.Date{Year: int32(year), Month: int32(month), Day: int32(day)}
	}

	return &pb.GetStatsReply{
		UserId:               totals.UserID,
		Xp:                   totals.XP,
		TotalAnswers:         totals.TotalAnswers,
		CorrectAnswers:       totals.CorrectAnswers,
		UniqueSolved:         totals.UniqueSolved,
		ActiveDays:           int32(totals.ActiveDays),
		CurrentStreak:        int64(totals.CurrentStreak),
		MaxStreak:            int64(totals.MaxStreak),
		CurrentCorrectStreak: int64(totals.CurrentCorrectStreak),
		BestCorrectStreak:    int64(totals.BestCorrectStreak),
		LastSolvedOn:         lastSolved,
	}, nil
}

func (s *Handler) GetXP(ctx context.Context, req *pb.UserStatsRequest) (*pb.GetXPReply, error) {
	xp, err := s.service.GetXP(ctx, req.GetUserId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.GetXPReply{Xp: xp}, nil
}

func (s *Handler) GetCurrentStreak(ctx context.Context, req *pb.UserStatsRequest) (*pb.GetCurrentStreakReply, error) {
	streak, err := s.service.GetCurrentStreak(ctx, req.GetUserId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.GetCurrentStreakReply{CurrentStreak: streak}, nil
}

func (s *Handler) GetCurrentCorrectStreak(ctx context.Context, req *pb.UserStatsRequest) (*pb.GetCurrentCorrectStreakReply, error) {
	streak, err := s.service.GetCurrentCorrectStreak(ctx, req.GetUserId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.GetCurrentCorrectStreakReply{CurrentCorrectStreak: streak}, nil
}

func toGRPCError(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "request timed out")
	case errors.Is(err, pgx.ErrNoRows):
		return status.Error(codes.NotFound, "user stats not found")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
