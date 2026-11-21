package stats_handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidStatsRequest     = errors.New("invalid stats request")
	ErrStatsNotFound           = errors.New("user stats not found")
	ErrServiceUnavailable      = errors.New("service unavailable")
	ErrGetStats                = errors.New("get stats error")
	ErrGetXP                   = errors.New("get xp error")
	ErrGetCurrentStreak        = errors.New("get current streak error")
	ErrGetCurrentCorrectStreak = errors.New("get current correct streak error")
)

func decodeGRPCError(err error) error {
	var mapped error
	switch {
	case errors.Is(err, context.Canceled):
		mapped = context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		mapped = context.DeadlineExceeded
	default:
		switch status.Code(err) {
		case codes.Canceled:
			mapped = context.Canceled
		case codes.DeadlineExceeded:
			mapped = context.DeadlineExceeded
		case codes.InvalidArgument:
			mapped = ErrInvalidStatsRequest
		case codes.NotFound:
			mapped = ErrStatsNotFound
		case codes.Unavailable:
			mapped = ErrServiceUnavailable
		}
	}
	if mapped == nil {
		return err
	}
	return fmt.Errorf("%w: %w", mapped, err)
}

func (h *Handler) handleServiceError(w http.ResponseWriter, r *http.Request, err error, fallback error) {
	status, message := http.StatusInternalServerError, fallback.Error()
	switch {
	case errors.Is(err, context.Canceled):
		status, message = http.StatusRequestTimeout, "request canceled"
	case errors.Is(err, context.DeadlineExceeded):
		status, message = http.StatusGatewayTimeout, "request timed out"
	case errors.Is(err, ErrInvalidStatsRequest):
		status, message = http.StatusBadRequest, ErrInvalidStatsRequest.Error()
	case errors.Is(err, ErrStatsNotFound):
		status, message = http.StatusNotFound, ErrStatsNotFound.Error()
	case errors.Is(err, ErrServiceUnavailable):
		status, message = http.StatusServiceUnavailable, ErrServiceUnavailable.Error()
	}
	if status >= http.StatusInternalServerError {
		h.logger.ErrorContext(r.Context(), "stats request failed",
			"method", r.Method,
			"path", r.URL.Path,
			"error", err,
		)
	}
	h.writeJSON(w, status, map[string]string{"error": message})
}
