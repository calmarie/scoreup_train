package user_handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	pb "github.com/calmarie/scoreup_train/backend/gen/user"
)

type Handler struct {
	client pb.UserClient
	logger *slog.Logger
}

func NewHandler(client pb.UserClient, logger *slog.Logger) *Handler {
	return &Handler{client: client, logger: logger}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/users/telegram/{telegramID}", h.createUser)
	mux.HandleFunc("GET /api/v1/users/telegram/{telegramID}", h.getUserByTelegramID)
}

func (h *Handler) getUserByTelegramID(w http.ResponseWriter, r *http.Request) {

	telegramID, err := strconv.ParseInt(r.PathValue("telegramID"), 10, 64)

	if err != nil || telegramID <= 0 {
		h.handleServiceError(w, r, ErrInvalidTelegramID, ErrGetUser)
		return
	}

	req := &pb.GetUserByTelegramIDRequest{
		TelegramId: telegramID,
	}

	reply, err := h.client.GetUserByTelegramID(r.Context(), req)
	if err != nil {
		h.handleServiceError(w, r, decodeGRPCError(err), ErrGetUser)
		return
	}

	user := User{
		ID:         reply.Id,
		TelegramID: reply.TelegramId,
		CreatedAt:  reply.CreatedAt.AsTime(),
	}

	h.writeJSON(w, http.StatusOK, user)

}

// createUser reads the Telegram ID from the URL and returns HTTP 409 if it is already registered.
func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {

	telegramID, err := strconv.ParseInt(r.PathValue("telegramID"), 10, 64)

	if err != nil || telegramID <= 0 {
		h.handleServiceError(w, r, ErrInvalidTelegramID, ErrCreateUser)
		return
	}

	req := &pb.CreateUserRequest{
		TelegramId: telegramID,
	}

	reply, err := h.client.CreateUser(r.Context(), req)
	if err != nil {
		h.handleServiceError(w, r, decodeGRPCError(err), ErrCreateUser)
		return
	}

	user := User{
		ID:         reply.Id,
		TelegramID: reply.TelegramId,
		CreatedAt:  reply.CreatedAt.AsTime(),
	}

	h.writeJSON(w, http.StatusCreated, user)

}

func (h *Handler) writeError(w http.ResponseWriter, statusCode int, message string) {
	h.writeJSON(w, statusCode, map[string]string{"error": message})
}

func (h *Handler) writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		h.logger.Error("write JSON response", "error", err)
	}
}

func (h *Handler) handleServiceError(w http.ResponseWriter, r *http.Request, err error, fallback error) {
	status, message := http.StatusInternalServerError, fallback.Error()
	switch {
	case errors.Is(err, context.Canceled):
		status, message = http.StatusRequestTimeout, "request canceled"
	case errors.Is(err, context.DeadlineExceeded):
		status, message = http.StatusGatewayTimeout, "request timed out"
	case errors.Is(err, ErrInvalidTelegramID):
		status, message = http.StatusBadRequest, ErrInvalidTelegramID.Error()
	case errors.Is(err, ErrUserNotFound):
		status, message = http.StatusNotFound, ErrUserNotFound.Error()
	case errors.Is(err, ErrUserAlreadyExists):
		status, message = http.StatusConflict, ErrUserAlreadyExists.Error()
	case errors.Is(err, ErrServiceUnavailable):
		status, message = http.StatusServiceUnavailable, ErrServiceUnavailable.Error()
	case errors.Is(err, ErrGetUser):
		status, message = http.StatusInternalServerError, ErrGetUser.Error()
	case errors.Is(err, ErrCreateUser):
		status, message = http.StatusInternalServerError, ErrCreateUser.Error()
	}

	if status >= http.StatusInternalServerError {
		h.logger.ErrorContext(r.Context(), "user service error",
			"method", r.Method,
			"path", r.URL.Path,
			"error", err,
		)
	}
	h.writeJSON(w, status, map[string]string{"error": message})
}
