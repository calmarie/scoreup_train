package question

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeQuestionService struct {
	question Question
	err      error
}

func (f *fakeQuestionService) GetQuestion(ctx context.Context) (Question, error) {
	return f.question, f.err
}

func (f *fakeQuestionService) SaveAnswer(ctx context.Context, answer SubmitAnswerRequest, userID int64) (Answer, error) {
	return Answer{}, nil
}

type answerRepository struct {
	saved Answer
}

func (r *answerRepository) GetQuestion(context.Context) (Question, error) {
	return Question{}, nil
}

func (r *answerRepository) GetCorrectAnswer(context.Context, int) (json.RawMessage, error) {
	return json.RawMessage(`"yes"`), nil
}

func (r *answerRepository) SaveAnswer(_ context.Context, answer Answer) (Answer, error) {
	r.saved = answer
	return answer, nil
}

func TestHandler_saveAnswerUsesResolvedUserID(t *testing.T) {
	repo := &answerRepository{}
	var resolvedTelegramID int64
	resolveUserID := func(_ context.Context, telegramID int64) (int64, error) {
		resolvedTelegramID = telegramID
		return 42, nil
	}
	h := NewHandler(NewService(repo), resolveUserID, slog.New(slog.NewTextHandler(io.Discard, nil)))
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/question/answer", strings.NewReader(`{"question_id":5,"answer":"yes"}`))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if resolvedTelegramID != 1 {
		t.Errorf("resolved Telegram ID = %d, want 1", resolvedTelegramID)
	}
	if repo.saved.User_ID != 42 {
		t.Errorf("saved user ID = %d, want 42", repo.saved.User_ID)
	}
	var answer Answer
	if err := json.NewDecoder(response.Body).Decode(&answer); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if answer.User_ID != 42 {
		t.Errorf("response user ID = %d, want 42", answer.User_ID)
	}
}

func TestHandler_getQuestion_Success(t *testing.T) {

	logger := slog.New(
		slog.NewTextHandler(io.Discard, nil),
	)

	service := &fakeQuestionService{
		question: Question{
			ID:   1,
			Text: "test question",
		},
	}

	h := NewHandler(service, nil, logger)

	r := httptest.NewRequest(http.MethodGet, "/question", nil)
	w := httptest.NewRecorder()

	h.getQuestion(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
	}

}

func TestHandler_getQuestion_Errors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
		message    string
	}{
		{
			name:       "question not found",
			err:        ErrQuestionNotFound,
			statusCode: http.StatusNotFound,
			message:    "Question not found",
		},
		{
			name:       "invalid request body",
			err:        ErrInvalidRequestBody,
			statusCode: http.StatusBadRequest,
			message:    "invalid request body",
		},
		{
			name:       "multiple JSON values",
			err:        ErrMultipleJSONValues,
			statusCode: http.StatusBadRequest,
			message:    "request body must contain one JSON object",
		},
		{
			name:       "request body too large",
			err:        ErrRequestBodyTooLarge,
			statusCode: http.StatusRequestEntityTooLarge,
			message:    "request body must not exceed 1 MiB",
		},
		{
			name:       "database unavailable",
			err:        ErrDatabaseUnavailable,
			statusCode: http.StatusServiceUnavailable,
			message:    "database temporarily unavailable",
		},
		{
			name:       "internal server error",
			err:        ErrInternalServer,
			statusCode: http.StatusInternalServerError,
			message:    "internal server error",
		},
		{
			name:       "request canceled",
			err:        context.Canceled,
			statusCode: http.StatusRequestTimeout,
			message:    "request canceled",
		},
		{
			name:       "request timed out",
			err:        context.DeadlineExceeded,
			statusCode: http.StatusGatewayTimeout,
			message:    "request timed out",
		},
	}

	logger := slog.New(
		slog.NewTextHandler(io.Discard, nil),
	)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &fakeQuestionService{
				err: test.err,
			}

			h := NewHandler(service, nil, logger)

			r := httptest.NewRequest(http.MethodGet, "/question", nil)
			w := httptest.NewRecorder()

			h.getQuestion(w, r)

			if w.Code != test.statusCode {
				t.Errorf("got status %d, want %d", w.Code, test.statusCode)
			}

			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("content type = %q, want %q", w.Header().Get("Content-Type"), "application/json")
			}

			var response map[string]string
			err := json.NewDecoder(w.Body).Decode(&response)
			if err != nil {
				t.Fatalf("decode response: %v", err)
			}

			if response["error"] != test.message {
				t.Errorf("error = %q, want %q", response["error"], test.message)
			}
		})
	}
}
