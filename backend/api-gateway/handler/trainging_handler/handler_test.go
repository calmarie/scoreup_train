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

func TestGetQuestionReturnsQuestionJSON(t *testing.T) {
	client := testTrainingClient{get: func(context.Context, *trainingpb.GetQuestionRequest) (*trainingpb.GetQuestionReply, error) {
		return &trainingpb.GetQuestionReply{
			Id: 5, Type: "radio", Difficulty: 2, Topic: "Go", Text: "Pick one",
			Options: []byte(`["a","b"]`), Data: []byte(`{"hint":true}`), IsActive: true,
		}, nil
	}}
	recorder := serveTrainingRoute(t, client, 0, http.MethodGet, "/api/v1/question", "")
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
			recorder := serveTrainingRoute(t, client, 0, http.MethodGet, "/api/v1/question", "")
			if recorder.Code != tt.want {
				t.Errorf("status = %d, want %d", recorder.Code, tt.want)
			}
			if strings.Contains(recorder.Body.String(), "private upstream detail") {
				t.Errorf("upstream detail leaked: %s", recorder.Body.String())
			}
		})
	}
}

func TestSaveAnswerUsesContextUserIDAndReturnsResult(t *testing.T) {
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	trainingClient := testTrainingClient{save: func(_ context.Context, req *trainingpb.SaveAnswerRequest) (*trainingpb.SaveAnswerReply, error) {
		if req.UserId != 77 || req.QuestionId != 5 || string(req.AnswerJson) != `"a"` {
			t.Errorf("save request = %+v, want user 77, question 5, answer a", req)
		}
		return &trainingpb.SaveAnswerReply{Id: 88, IsCorrect: true, CreatedAt: timestamppb.New(createdAt)}, nil
	}}
	recorder := serveTrainingRoute(t, trainingClient, 77, http.MethodPost, "/api/v1/question/answer", `{"question_id":5,"answer":"a"}`)
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
	for _, body := range []string{`{`, `{}`, `{"user_id":123,"question_id":5,"answer":"a"}`, `{"question_id":0,"answer":"a"}`, `{"question_id":5}`} {
		recorder := serveTrainingRoute(t, trainingClient, 77, http.MethodPost, "/api/v1/question/answer", body)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("body %q: status = %d, want 400", body, recorder.Code)
		}
	}
}

func TestSaveAnswerRejectsMissingContextUserID(t *testing.T) {
	trainingClient := testTrainingClient{save: func(context.Context, *trainingpb.SaveAnswerRequest) (*trainingpb.SaveAnswerReply, error) {
		t.Fatal("training RPC called without an authenticated user")
		return nil, nil
	}}
	recorder := serveTrainingRoute(t, trainingClient, 0, http.MethodPost, "/api/v1/question/answer", `{"question_id":5,"answer":"a"}`)
	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, body = %s; want 401", recorder.Code, recorder.Body.String())
	}
}

func TestSaveAnswerReturnsNotFoundWhenQuestionIsMissing(t *testing.T) {
	trainingClient := testTrainingClient{save: func(context.Context, *trainingpb.SaveAnswerRequest) (*trainingpb.SaveAnswerReply, error) {
		return nil, status.Error(codes.NotFound, "private upstream detail")
	}}
	recorder := serveTrainingRoute(t, trainingClient, 77, http.MethodPost, "/api/v1/question/answer", `{"question_id":5,"answer":"a"}`)
	if recorder.Code != http.StatusNotFound || !strings.Contains(recorder.Body.String(), "question not found") {
		t.Errorf("status = %d, body = %s; want question not found", recorder.Code, recorder.Body.String())
	}
}

func serveTrainingRoute(t *testing.T, trainingClient trainingpb.TrainingClient, userID int64, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewHandler(trainingClient, logger)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if userID > 0 {
		req = req.WithContext(context.WithValue(req.Context(), "userID", userID))
	}
	mux.ServeHTTP(recorder, req)
	return recorder
}
