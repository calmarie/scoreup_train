package handler

import (
	"context"
	"errors"
	"time"

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
		AccuracyPercent:      totals.AccuracyPercent,
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

func (s *Handler) GetDailyStats(ctx context.Context, req *pb.GetDailyStatsRequest) (*pb.DailyStats, error) {
	if req.GetUserId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	year, month, day := int(req.GetDate().GetYear()), time.Month(req.GetDate().GetMonth()), int(req.GetDate().GetDay())
	requestedDay := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	if year < 1 || year > 9999 || requestedDay.Year() != year || requestedDay.Month() != month || requestedDay.Day() != day {
		return nil, status.Error(codes.InvalidArgument, "invalid date")
	}

	daily, err := s.service.GetDailyStats(ctx, req.GetUserId(), requestedDay)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return dailyStatsReply(daily), nil
}

func (s *Handler) GetWeeklyStats(ctx context.Context, req *pb.UserStatsRequest) (*pb.GetWeeklyStatsReply, error) {
	if req.GetUserId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	weekly, err := s.service.GetWeeklyStats(ctx, req.GetUserId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	days := make([]*pb.DailyStats, 0, len(weekly))
	for _, daily := range weekly {
		days = append(days, dailyStatsReply(daily))
	}
	return &pb.GetWeeklyStatsReply{Days: days}, nil
}

func (s *Handler) GetTopicStats(ctx context.Context, req *pb.GetTopicStatsRequest) (*pb.TopicStats, error) {
	if req.GetUserId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	if req.GetTopic() == "" {
		return nil, status.Error(codes.InvalidArgument, "topic is required")
	}
	topic, err := s.service.GetTopicStats(ctx, req.GetUserId(), req.GetTopic())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return topicStatsReply(topic), nil
}

func (s *Handler) GetAllTopicStats(ctx context.Context, req *pb.UserStatsRequest) (*pb.GetAllTopicStatsReply, error) {
	if req.GetUserId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	allTopics, err := s.service.GetAllTopicStats(ctx, req.GetUserId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	topics := make([]*pb.TopicStats, 0, len(allTopics))
	for _, topic := range allTopics {
		topics = append(topics, topicStatsReply(topic))
	}
	return &pb.GetAllTopicStatsReply{Topics: topics}, nil
}

func dailyStatsReply(daily stats.UserDailyStats) *pb.DailyStats {
	year, month, day := daily.Date.UTC().Date()
	return &pb.DailyStats{
		Date:     &date.Date{Year: int32(year), Month: int32(month), Day: int32(day)},
		Attempts: daily.Attempts,
		Correct:  daily.Correct,
		Solved:   daily.Solved,
		Xp:       daily.XP,
	}
}

func topicStatsReply(topic stats.UserTopicStats) *pb.TopicStats {
	return &pb.TopicStats{
		Topic:    topic.Topic,
		Attempts: topic.Attempts,
		Correct:  topic.Correct,
		Solved:   topic.Solved,
		Xp:       topic.XP,
	}
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
