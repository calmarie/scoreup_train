package users

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
)

type Handler struct {
	service UserService
	logger  *slog.Logger
}

func NewHandler(service UserService, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/users", h.createUser)
	mux.HandleFunc("GET /api/v1/users/telegram/{telegramID}", h.getUserByTelegramID)
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	user, err := h.service.CreateUser(r.Context())
	if err != nil {
		h.handleServiceError(w, r, err, ErrCreateUser)
		return
	}

	h.writeJSON(w, http.StatusCreated, user)
}

func (h *Handler) getUserByTelegramID(w http.ResponseWriter, r *http.Request) {
	telegramID, err := strconv.ParseInt(r.PathValue("telegramID"), 10, 64)
	if err != nil {
		h.handleServiceError(w, r, ErrInvalidTelegramID, ErrGetUser)
		return
	}

	user, err := h.service.GetUserByTelegramID(r.Context(), telegramID)
	if err != nil {
		h.handleServiceError(w, r, err, ErrGetUser)
		return
	}

	h.writeJSON(w, http.StatusOK, user)
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
	case errors.Is(err, ErrDatabaseUnavailable):
		status, message = http.StatusServiceUnavailable, ErrDatabaseUnavailable.Error()
	case errors.Is(err, ErrGetUser):
		status, message = http.StatusInternalServerError, ErrGetUser.Error()
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

func (h *Handler) writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		h.logger.Error("write JSON response", "error", err)
	}
}
