package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	pb "github.com/calmarie/scoreup_train/backend/gen/training"
	"github.com/calmarie/scoreup_train/backend/training-service/internal/question"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type questionRepositoryForGRPC struct {
	value question.Question
	err   error
}

func (r questionRepositoryForGRPC) GetQuestion(context.Context) (question.Question, error) {
	return r.value, r.err
}

func (questionRepositoryForGRPC) SaveAnswer(context.Context, question.Answer) (question.Answer, error) {
	return question.Answer{}, errors.New("unexpected SaveAnswer call")
}

func (questionRepositoryForGRPC) GetCorrectAnswer(context.Context, int) (json.RawMessage, error) {
	return nil, errors.New("unexpected GetCorrectAnswer call")
}

func TestGRPCGetQuestionMapsServiceResult(t *testing.T) {
	skill := "arithmetic"
	imageURL := "https://example.com/image.png"
	explanation := "add the numbers"
	repo := questionRepositoryForGRPC{value: question.Question{
		ID: 12, Type: "radio", Difficulty: 3, Topic: "math", Skill: &skill,
		Text: "2 + 2?", ImageURL: &imageURL, Options: json.RawMessage(`["3","4"]`),
		Explanation: &explanation, Data: json.RawMessage(`{"source":"test"}`), IsActive: true, XP: 75,
	}}
	srv := &trainingGRPCserver{service: question.NewService(repo)}

	got, err := srv.GetQuestion(context.Background(), &pb.GetQuestionRequest{})
	if err != nil {
		t.Fatalf("GetQuestion: %v", err)
	}
	want := &pb.GetQuestionReply{
		Id: 12, Type: "radio", Difficulty: 3, Topic: "math", Skill: &skill,
		Text: "2 + 2?", ImageUrl: &imageURL, Options: []byte(`["3","4"]`),
		Explanation: &explanation, Data: []byte(`{"source":"test"}`), IsActive: true, Xp: 75,
	}
	if !proto.Equal(got, want) {
		t.Fatalf("GetQuestion() = %v, want %v", got, want)
	}
}

func TestGRPCGetQuestionMapsNotFound(t *testing.T) {
	srv := &trainingGRPCserver{service: question.NewService(questionRepositoryForGRPC{err: question.ErrQuestionNotFound})}

	_, err := srv.GetQuestion(context.Background(), &pb.GetQuestionRequest{})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("GetQuestion() code = %v, want %v", status.Code(err), codes.NotFound)
	}
}

type answerRepositoryForGRPC struct {
	saved     question.Answer
	createdAt time.Time
}

func (answerRepositoryForGRPC) GetQuestion(context.Context) (question.Question, error) {
	return question.Question{}, errors.New("unexpected GetQuestion call")
}

func (answerRepositoryForGRPC) GetCorrectAnswer(context.Context, int) (json.RawMessage, error) {
	return json.RawMessage(`"yes"`), nil
}

func (r *answerRepositoryForGRPC) SaveAnswer(_ context.Context, answer question.Answer) (question.Answer, error) {
	r.saved = answer
	answer.ID = 19
	answer.XP = 75
	answer.CreatedAt = r.createdAt
	return answer, nil
}

func TestGRPCSaveAnswerMapsRequestAndTimestamp(t *testing.T) {
	createdAt := time.Date(2026, time.November, 14, 12, 30, 45, 123456000, time.UTC)
	repo := &answerRepositoryForGRPC{createdAt: createdAt}
	srv := &trainingGRPCserver{service: question.NewService(repo)}
	request := &pb.SaveAnswerRequest{UserId: 7, QuestionId: 3, AnswerJson: []byte(`"yes"`)}

	got, err := srv.SaveAnswer(context.Background(), request)
	if err != nil {
		t.Fatalf("SaveAnswer: %v", err)
	}
	want := &pb.SaveAnswerReply{Id: 19, IsCorrect: true, CreatedAt: timestamppb.New(createdAt), Xp: 75}
	if !proto.Equal(got, want) {
		t.Fatalf("SaveAnswer() = %v, want %v", got, want)
	}
	if repo.saved.UserID != 7 || repo.saved.QuestionID != 3 || string(repo.saved.Answer) != `"yes"` {
		t.Fatalf("saved answer = %+v, want user 7, question 3, answer yes", repo.saved)
	}
}
