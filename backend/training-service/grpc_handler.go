package main

import (
	"context"
	"errors"

	pb "github.com/calmarie/scoreup_train/backend/gen/training"
	"github.com/calmarie/scoreup_train/backend/training-service/internal/question"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *trainingGRPCserver) GetQuestion(
	ctx context.Context,
	req *pb.GetQuestionRequest,
) (*pb.GetQuestionReply, error) {

	q, err := s.service.GetQuestion(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.GetQuestionReply{
		Id:          int32(q.ID),
		Type:        q.Type,
		Difficulty:  int32(q.Difficulty),
		Topic:       q.Topic,
		Skill:       q.Skill,
		Text:        q.Text,
		ImageUrl:    q.ImageURL,
		Options:     q.Options,
		Explanation: q.Explanation,
		Data:        q.Data,
		IsActive:    q.IsActive,
		Xp:          int32(q.XP),
	}, nil
}

// SaveAnswer returns the question's configured XP even for incorrect or repeated answers.
func (s *trainingGRPCserver) SaveAnswer(
	ctx context.Context,
	req *pb.SaveAnswerRequest,
) (*pb.SaveAnswerReply, error) {

	submit := question.SubmitAnswerRequest{
		UserID:     req.UserId,
		QuestionID: int(req.QuestionId),
		Answer:     req.AnswerJson,
	}

	ans, err := s.service.SaveAnswer(ctx, submit)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.SaveAnswerReply{
		Id:         ans.ID,
		IsCorrect:  ans.IsCorrect,
		CreatedAt:  timestamppb.New(ans.CreatedAt),
		Xp:         int32(ans.XP),
		FirstSolve: ans.FirstSolve,
	}, nil
}

func toGRPCError(err error) error {
	switch {
	case errors.Is(err, question.ErrPublishAttempt):
		return status.Error(codes.Internal, question.ErrPublishAttempt.Error())
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "request timed out")
	case errors.Is(err, question.ErrQuestionNotFound):
		return status.Error(codes.NotFound, "question not found")
	case errors.Is(err, question.ErrDatabaseUnavailable):
		return status.Error(codes.Unavailable, "database unavailable")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
