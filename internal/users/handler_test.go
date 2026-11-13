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

func (f *fakeUserRepository) GetUserByTelegramID(ctx context.Context, telegramID int64) (User, error) {
	return User{}, f.err
}

type fakeUserService struct {
	user                User
	err                 error
	requestedTelegramID int64
	getUserCalled       bool
}

func (f *fakeUserService) CreateUser(ctx context.Context) (User, error) {
	return f.user, f.err
}

func (f *fakeUserService) GetUserByTelegramID(ctx context.Context, telegramID int64) (User, error) {
	f.requestedTelegramID = telegramID
	f.getUserCalled = true
	return f.user, f.err
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

func TestHandler_getUserByTelegramID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("success", func(t *testing.T) {
		telegramID := int64(123456)
		service := &fakeUserService{
			user: User{Id: 1, Telegram_id: &telegramID},
		}
		h := NewHandler(service, logger)
		mux := http.NewServeMux()
		h.RegisterRoutes(mux)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/users/telegram/123456", nil)
		mux.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
		}
		if !service.getUserCalled {
			t.Fatal("GetUserByTelegramID was not called")
		}
		if service.requestedTelegramID != telegramID {
			t.Fatalf("telegram ID = %d, want %d", service.requestedTelegramID, telegramID)
		}

		var response User
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if response.Id != 1 {
			t.Errorf("id = %d, want 1", response.Id)
		}
		if response.Telegram_id == nil || *response.Telegram_id != telegramID {
			t.Errorf("telegram_id = %v, want %d", response.Telegram_id, telegramID)
		}
	})

	for _, telegramID := range []string{"not-a-number", "9223372036854775808"} {
		t.Run("invalid ID "+telegramID, func(t *testing.T) {
			service := &fakeUserService{}
			h := NewHandler(service, logger)
			mux := http.NewServeMux()
			h.RegisterRoutes(mux)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/api/v1/users/telegram/"+telegramID, nil)
			mux.ServeHTTP(w, r)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
			if service.getUserCalled {
				t.Fatal("GetUserByTelegramID was called for an invalid Telegram ID")
			}
		})
	}

	t.Run("not found", func(t *testing.T) {
		service := &fakeUserService{err: ErrUserNotFound}
		h := NewHandler(service, logger)
		mux := http.NewServeMux()
		h.RegisterRoutes(mux)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/users/telegram/123456", nil)
		mux.ServeHTTP(w, r)

		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
		var response struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if response.Error != ErrUserNotFound.Error() {
			t.Errorf("error = %q, want %q", response.Error, ErrUserNotFound.Error())
		}
	})

	t.Run("service error", func(t *testing.T) {
		service := &fakeUserService{err: errors.New("unexpected service error")}
		h := NewHandler(service, logger)
		mux := http.NewServeMux()
		h.RegisterRoutes(mux)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/users/telegram/123456", nil)
		mux.ServeHTTP(w, r)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
		var response struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if response.Error != ErrGetUser.Error() {
			t.Errorf("error = %q, want %q", response.Error, ErrGetUser.Error())
		}
	})
}
