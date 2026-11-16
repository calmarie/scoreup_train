package user_handler

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

	pb "github.com/calmarie/scoreup_train/backend/gen/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type testUserClient struct {
	pb.UserClient
	get    func(context.Context, *pb.GetUserByTelegramIDRequest) (*pb.UserReply, error)
	create func(context.Context, *pb.CreateUserRequest) (*pb.UserReply, error)
}

func (c testUserClient) GetUserByTelegramID(ctx context.Context, req *pb.GetUserByTelegramIDRequest, _ ...grpc.CallOption) (*pb.UserReply, error) {
	return c.get(ctx, req)
}

func (c testUserClient) CreateUser(ctx context.Context, req *pb.CreateUserRequest, _ ...grpc.CallOption) (*pb.UserReply, error) {
	return c.create(ctx, req)
}

func TestCreateUserReturnsCreatedUser(t *testing.T) {
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	client := testUserClient{create: func(_ context.Context, req *pb.CreateUserRequest) (*pb.UserReply, error) {
		if req.TelegramId != 123 {
			t.Errorf("telegram ID = %d, want 123", req.TelegramId)
		}
		return &pb.UserReply{Id: 42, TelegramId: 123, CreatedAt: timestamppb.New(createdAt)}, nil
	}}
	recorder := serveUserRoute(t, client, http.MethodPost, "123")
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", recorder.Code, recorder.Body.String())
	}
	var user User
	if err := json.Unmarshal(recorder.Body.Bytes(), &user); err != nil {
		t.Fatal(err)
	}
	if user.ID != 42 || user.TelegramID != 123 || !user.CreatedAt.Equal(createdAt) {
		t.Errorf("user = %+v, want ID 42, TelegramID 123, CreatedAt %v", user, createdAt)
	}
}

func TestCreateUserRejectsInvalidTelegramID(t *testing.T) {
	client := testUserClient{create: func(context.Context, *pb.CreateUserRequest) (*pb.UserReply, error) {
		t.Fatal("gRPC client called for invalid telegram ID")
		return nil, nil
	}}
	for _, id := range []string{"abc", "0", "-12"} {
		t.Run(id, func(t *testing.T) {
			recorder := serveUserRoute(t, client, http.MethodPost, id)
			if recorder.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", recorder.Code)
			}
		})
	}
}

func TestCreateUserMapsGRPCErrors(t *testing.T) {
	tests := []struct {
		name string
		code codes.Code
		want int
	}{
		{"already exists", codes.AlreadyExists, http.StatusConflict},
		{"unavailable", codes.Unavailable, http.StatusServiceUnavailable},
		{"internal", codes.Internal, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := testUserClient{create: func(context.Context, *pb.CreateUserRequest) (*pb.UserReply, error) {
				return nil, status.Error(tt.code, "private upstream detail")
			}}
			recorder := serveUserRoute(t, client, http.MethodPost, "123")
			if recorder.Code != tt.want {
				t.Errorf("status = %d, want %d; body = %s", recorder.Code, tt.want, recorder.Body.String())
			}
			if strings.Contains(recorder.Body.String(), "private upstream detail") {
				t.Errorf("upstream detail leaked: %q", recorder.Body.String())
			}
		})
	}
}

func TestGetUserByTelegramIDReturnsUser(t *testing.T) {
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	client := testUserClient{get: func(_ context.Context, req *pb.GetUserByTelegramIDRequest) (*pb.UserReply, error) {
		if req.TelegramId != 123 {
			t.Errorf("telegram ID = %d, want 123", req.TelegramId)
		}
		return &pb.UserReply{Id: 42, TelegramId: 123, CreatedAt: timestamppb.New(createdAt), XpTotal: 150}, nil
	}}
	recorder := serveGetUser(t, client, "123")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", recorder.Code, recorder.Body.String())
	}
	var user User
	if err := json.Unmarshal(recorder.Body.Bytes(), &user); err != nil {
		t.Fatal(err)
	}
	if user.ID != 42 || user.TelegramID != 123 || !user.CreatedAt.Equal(createdAt) || user.XPTotal != 150 {
		t.Errorf("user = %+v, want ID 42, TelegramID 123, CreatedAt %v", user, createdAt)
	}
}

func TestGetUserByTelegramIDRejectsInvalidPath(t *testing.T) {
	client := testUserClient{get: func(context.Context, *pb.GetUserByTelegramIDRequest) (*pb.UserReply, error) {
		t.Fatal("gRPC client called for invalid telegram ID")
		return nil, nil
	}}
	for _, pathID := range []string{"abc", "0", "-12"} {
		t.Run(pathID, func(t *testing.T) {
			recorder := serveGetUser(t, client, pathID)
			if recorder.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", recorder.Code)
			}
		})
	}
}

func TestGetUserByTelegramIDMapsGRPCErrors(t *testing.T) {
	tests := []struct {
		name string
		code codes.Code
		want int
	}{
		{"not found", codes.NotFound, http.StatusNotFound},
		{"invalid argument", codes.InvalidArgument, http.StatusBadRequest},
		{"unavailable", codes.Unavailable, http.StatusServiceUnavailable},
		{"deadline exceeded", codes.DeadlineExceeded, http.StatusGatewayTimeout},
		{"internal", codes.Internal, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := testUserClient{get: func(context.Context, *pb.GetUserByTelegramIDRequest) (*pb.UserReply, error) {
				return nil, status.Error(tt.code, "private upstream detail")
			}}
			recorder := serveGetUser(t, client, "123")
			if recorder.Code != tt.want {
				t.Errorf("status = %d, want %d; body = %s", recorder.Code, tt.want, recorder.Body.String())
			}
			if recorder.Body.String() == "" || strings.Contains(recorder.Body.String(), "private upstream detail") {
				t.Errorf("unsafe or empty error body: %q", recorder.Body.String())
			}
		})
	}
}

func serveGetUser(t *testing.T, client pb.UserClient, pathID string) *httptest.ResponseRecorder {
	return serveUserRoute(t, client, http.MethodGet, pathID)
}

func serveUserRoute(t *testing.T, client pb.UserClient, method, pathID string) *httptest.ResponseRecorder {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewHandler(client, logger)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	request := httptest.NewRequest(method, "/api/v1/users/telegram/"+pathID, nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	return recorder
}
