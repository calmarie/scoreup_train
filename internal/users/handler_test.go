package users

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeUserRepository struct {
	err error
}

func (f *fakeUserRepository) CreateUser(ctx context.Context) (User, error) {
	return User{Id: 1}, f.err
}

func TestHandler_createUser(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		status  int
		message string
	}{
		{"success", nil, http.StatusCreated, ""},
		{"duplicate", fmt.Errorf("insert: %w", ErrUserAlreadyExists), http.StatusConflict, "user already exists"},
		{"database unavailable", ErrDatabaseUnavailable, http.StatusServiceUnavailable, "database temporarily unavailable"},
		{"create error", ErrCreateUser, http.StatusInternalServerError, "create user error"},
		{"unknown error", errors.New("database details"), http.StatusInternalServerError, "create user error"},
		{"canceled", context.Canceled, http.StatusRequestTimeout, "request canceled"},
		{"timed out", context.DeadlineExceeded, http.StatusGatewayTimeout, "request timed out"},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := NewHandler(NewService(&fakeUserRepository{err: test.err}), logger)
			mux := http.NewServeMux()
			h.RegisterRoutes(mux)
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)
			mux.ServeHTTP(w, r)

			if w.Code != test.status {
				t.Errorf("status = %d, want %d", w.Code, test.status)
			}
			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("content type = %q, want application/json", w.Header().Get("Content-Type"))
			}

			var response struct {
				ID    int64  `json:"id"`
				Error string `json:"error"`
			}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response.Error != test.message {
				t.Errorf("error = %q, want %q", response.Error, test.message)
			}
			if test.err == nil && response.ID != 1 {
				t.Errorf("id = %d, want 1", response.ID)
			}
		})
	}
}
