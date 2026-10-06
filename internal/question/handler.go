package question

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

const maxRequestBodyBytes int64 = 1 << 20

type serviceErrorResponse struct {
	statusCode int
	message    string
}

type Handler struct {
	service QuestistionService
	logger  *slog.Logger
}

func NewHandler(service QuestistionService, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/question", h.getQuestion)
}

func (h *Handler) getQuestion(w http.ResponseWriter, r *http.Request) {

	order, err := h.service.GetQuestion(r.Context())
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusOK, order)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {

	//MaxBytesReader limits max size of http Body to defend server from server memory overload
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return requestBodyError(err)
	}

	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		if err != nil {
			return requestBodyError(err)
		}
		return ErrMultipleJSONValues
	}

	return nil
}

func requestBodyError(err error) error {
	var sizeErr *http.MaxBytesError
	if errors.As(err, &sizeErr) {
		return fmt.Errorf("%w: %w", ErrRequestBodyTooLarge, err)
	}
	return fmt.Errorf("%w: %w", ErrInvalidRequestBody, err)
}

func (h *Handler) handleServiceError(w http.ResponseWriter, r *http.Request, err error) {
	response := serviceErrorResponses[ErrInternalServer]
	// Cancellation takes precedence when a database error also wraps a context error.
	if errors.Is(err, context.Canceled) {
		response = serviceErrorResponses[context.Canceled]
	} else if errors.Is(err, context.DeadlineExceeded) {
		response = serviceErrorResponses[context.DeadlineExceeded]
	} else {
		for knownErr, knownResponse := range serviceErrorResponses {
			if errors.Is(err, knownErr) {
				response = knownResponse
				break
			}
		}
	}

	if response.statusCode >= http.StatusInternalServerError {
		h.logger.ErrorContext(r.Context(), "question service error",
			"method", r.Method, "path", r.URL.Path,
			"status", response.statusCode, "error", err)
	}
	h.writeError(w, response.statusCode, response.message)
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
