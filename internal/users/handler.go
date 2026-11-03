package users

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
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
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	user, err := h.service.CreateUser(r.Context())
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, user)
}

func (h *Handler) handleServiceError(w http.ResponseWriter, r *http.Request, err error) {
	status, message := http.StatusInternalServerError, ErrCreateUser.Error()
	switch {
	case errors.Is(err, context.Canceled):
		status, message = http.StatusRequestTimeout, "request canceled"
	case errors.Is(err, context.DeadlineExceeded):
		status, message = http.StatusGatewayTimeout, "request timed out"
	case errors.Is(err, ErrUserAlreadyExists):
		status, message = http.StatusConflict, ErrUserAlreadyExists.Error()
	case errors.Is(err, ErrDatabaseUnavailable):
		status, message = http.StatusServiceUnavailable, ErrDatabaseUnavailable.Error()
	}

	if status >= http.StatusInternalServerError {
		h.logger.ErrorContext(r.Context(), "create user", "error", err)
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
