package training_handler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	trainingpb "github.com/calmarie/scoreup_train/backend/gen/training"
	userpb "github.com/calmarie/scoreup_train/backend/gen/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type testTrainingClient struct {
	trainingpb.TrainingClient
	get  func(context.Context, *trainingpb.GetQuestionRequest) (*trainingpb.GetQuestionReply, error)
	save func(context.Context, *trainingpb.SaveAnswerRequest) (*trainingpb.SaveAnswerReply, error)
}

func (c testTrainingClient) GetQuestion(ctx context.Context, req *trainingpb.GetQuestionRequest, _ ...grpc.CallOption) (*trainingpb.GetQuestionReply, error) {
	return c.get(ctx, req)
}

func (c testTrainingClient) SaveAnswer(ctx context.Context, req *trainingpb.SaveAnswerRequest, _ ...grpc.CallOption) (*trainingpb.SaveAnswerReply, error) {
	return c.save(ctx, req)
}

type testUserClient struct {
	userpb.UserClient
	get func(context.Context, *userpb.GetUserByTelegramIDRequest) (*userpb.UserReply, error)
}

func (c testUserClient) GetUserByTelegramID(ctx context.Context, req *userpb.GetUserByTelegramIDRequest, _ ...grpc.CallOption) (*userpb.UserReply, error) {
	return c.get(ctx, req)
}

func TestGetQuestionReturnsQuestionJSON(t *testing.T) {
	client := testTrainingClient{get: func(context.Context, *trainingpb.GetQuestionRequest) (*trainingpb.GetQuestionReply, error) {
		return &trainingpb.GetQuestionReply{
			Id: 5, Type: "radio", Difficulty: 2, Topic: "Go", Text: "Pick one",
			Options: []byte(`["a","b"]`), Data: []byte(`{"hint":true}`), IsActive: true,
		}, nil
	}}
	recorder := serveTrainingRoute(t, client, nil, http.MethodGet, "/api/v1/question", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", recorder.Code, recorder.Body.String())
	}
	var question Question
	if err := json.Unmarshal(recorder.Body.Bytes(), &question); err != nil {
		t.Fatal(err)
	}
	if question.ID != 5 || question.Type != "radio" || question.Topic != "Go" || string(question.Options) != `["a","b"]` || string(question.Data) != `{"hint":true}` {
		t.Errorf("question = %+v", question)
	}
	if strings.Contains(recorder.Body.String(), "correct_answer") {
		t.Errorf("answer leaked in question response: %s", recorder.Body.String())
	}
}

func TestGetQuestionMapsGRPCErrors(t *testing.T) {
	for _, tt := range []struct {
		code codes.Code
		want int
	}{
		{codes.NotFound, http.StatusNotFound},
		{codes.Unavailable, http.StatusServiceUnavailable},
		{codes.Internal, http.StatusInternalServerError},
	} {
		t.Run(tt.code.String(), func(t *testing.T) {
			client := testTrainingClient{get: func(context.Context, *trainingpb.GetQuestionRequest) (*trainingpb.GetQuestionReply, error) {
				return nil, status.Error(tt.code, "private upstream detail")
			}}
			recorder := serveTrainingRoute(t, client, nil, http.MethodGet, "/api/v1/question", "")
			if recorder.Code != tt.want {
				t.Errorf("status = %d, want %d", recorder.Code, tt.want)
			}
			if strings.Contains(recorder.Body.String(), "private upstream detail") {
				t.Errorf("upstream detail leaked: %s", recorder.Body.String())
			}
		})
	}
}

func TestSaveAnswerResolvesUserAndReturnsResult(t *testing.T) {
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	userClient := testUserClient{get: func(_ context.Context, req *userpb.GetUserByTelegramIDRequest) (*userpb.UserReply, error) {
		if req.TelegramId != 123 {
			t.Errorf("telegram ID = %d, want 123", req.TelegramId)
		}
		return &userpb.UserReply{Id: 77}, nil
	}}
	trainingClient := testTrainingClient{save: func(_ context.Context, req *trainingpb.SaveAnswerRequest) (*trainingpb.SaveAnswerReply, error) {
		if req.UserId != 77 || req.QuestionId != 5 || string(req.AnswerJson) != `"a"` {
			t.Errorf("save request = %+v, want user 77, question 5, answer a", req)
		}
		return &trainingpb.SaveAnswerReply{Id: 88, IsCorrect: true, CreatedAt: timestamppb.New(createdAt)}, nil
	}}
	recorder := serveTrainingRoute(t, trainingClient, userClient, http.MethodPost, "/api/v1/question/answer", `{"telegram_id":123,"question_id":5,"answer":"a"}`)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", recorder.Code, recorder.Body.String())
	}
	var answer Answer
	if err := json.Unmarshal(recorder.Body.Bytes(), &answer); err != nil {
		t.Fatal(err)
	}
	if answer.ID != 88 || !answer.IsCorrect || !answer.CreatedAt.Equal(createdAt) {
		t.Errorf("answer = %+v, want ID 88, correct, created at %v", answer, createdAt)
	}
}

func TestSaveAnswerRejectsInvalidBodyBeforeRPC(t *testing.T) {
	trainingClient := testTrainingClient{save: func(context.Context, *trainingpb.SaveAnswerRequest) (*trainingpb.SaveAnswerReply, error) {
		t.Fatal("training RPC called for invalid body")
		return nil, nil
	}}
	userClient := testUserClient{get: func(context.Context, *userpb.GetUserByTelegramIDRequest) (*userpb.UserReply, error) {
		t.Fatal("user RPC called for invalid body")
		return nil, nil
	}}
	for _, body := range []string{`{`, `{}`, `{"telegram_id":0,"question_id":5,"answer":"a"}`, `{"telegram_id":123,"question_id":0,"answer":"a"}`, `{"telegram_id":123,"question_id":5}`} {
		recorder := serveTrainingRoute(t, trainingClient, userClient, http.MethodPost, "/api/v1/question/answer", body)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("body %q: status = %d, want 400", body, recorder.Code)
		}
	}
}

func TestSaveAnswerReturnsNotFoundWhenUserIsMissing(t *testing.T) {
	userClient := testUserClient{get: func(context.Context, *userpb.GetUserByTelegramIDRequest) (*userpb.UserReply, error) {
		return nil, status.Error(codes.NotFound, "private upstream detail")
	}}
	trainingClient := testTrainingClient{save: func(context.Context, *trainingpb.SaveAnswerRequest) (*trainingpb.SaveAnswerReply, error) {
		t.Fatal("training RPC called when user is missing")
		return nil, nil
	}}
	recorder := serveTrainingRoute(t, trainingClient, userClient, http.MethodPost, "/api/v1/question/answer", `{"telegram_id":123,"question_id":5,"answer":"a"}`)
	if recorder.Code != http.StatusNotFound || !strings.Contains(recorder.Body.String(), "user not found") {
		t.Errorf("status = %d, body = %s; want user not found", recorder.Code, recorder.Body.String())
	}
}

func TestSaveAnswerReturnsNotFoundWhenQuestionIsMissing(t *testing.T) {
	userClient := testUserClient{get: func(context.Context, *userpb.GetUserByTelegramIDRequest) (*userpb.UserReply, error) {
		return &userpb.UserReply{Id: 77}, nil
	}}
	trainingClient := testTrainingClient{save: func(context.Context, *trainingpb.SaveAnswerRequest) (*trainingpb.SaveAnswerReply, error) {
		return nil, status.Error(codes.NotFound, "private upstream detail")
	}}
	recorder := serveTrainingRoute(t, trainingClient, userClient, http.MethodPost, "/api/v1/question/answer", `{"telegram_id":123,"question_id":5,"answer":"a"}`)
	if recorder.Code != http.StatusNotFound || !strings.Contains(recorder.Body.String(), "question not found") {
		t.Errorf("status = %d, body = %s; want question not found", recorder.Code, recorder.Body.String())
	}
}

func serveTrainingRoute(t *testing.T, trainingClient trainingpb.TrainingClient, userClient userpb.UserClient, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewHandler(trainingClient, userClient, logger)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(method, path, strings.NewReader(body)))
	return recorder
}
