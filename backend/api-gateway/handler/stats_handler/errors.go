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
	ErrInvalidTelegramID       = errors.New("invalid telegram id")
	ErrInvalidStatsRequest     = errors.New("invalid stats request")
	ErrUserNotFound            = errors.New("user not found")
	ErrStatsNotFound           = errors.New("user stats not found")
	ErrServiceUnavailable      = errors.New("service unavailable")
	ErrGetUser                 = errors.New("get user error")
	ErrGetStats                = errors.New("get stats error")
	ErrGetXP                   = errors.New("get xp error")
	ErrGetCurrentStreak        = errors.New("get current streak error")
	ErrGetCurrentCorrectStreak = errors.New("get current correct streak error")
)

// decodeGRPCError maps lookup errors for the upstream service that returned them.
func decodeGRPCError(err, invalidArgument, notFound error) error {
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
			mapped = invalidArgument
		case codes.NotFound:
			mapped = notFound
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
	case errors.Is(err, ErrInvalidTelegramID):
		status, message = http.StatusBadRequest, ErrInvalidTelegramID.Error()
	case errors.Is(err, ErrInvalidStatsRequest):
		status, message = http.StatusBadRequest, ErrInvalidStatsRequest.Error()
	case errors.Is(err, ErrUserNotFound):
		status, message = http.StatusNotFound, ErrUserNotFound.Error()
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
