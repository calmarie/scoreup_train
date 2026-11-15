package training_handler

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidRequestBody  = errors.New("invalid request body")
	ErrMultipleJSONValues  = errors.New("request body must contain one JSON object")
	ErrRequestBodyTooLarge = errors.New("request body must not exceed 1 MiB")
	ErrInvalidTelegramID   = errors.New("invalid telegram id")
	ErrInvalidQuestionID   = errors.New("invalid question id")
	ErrAnswerRequired      = errors.New("answer is required")
	ErrUserNotFound        = errors.New("user not found")
	ErrQuestionNotFound    = errors.New("question not found")
	ErrServiceUnavailable  = errors.New("service unavailable")
	ErrGetUser             = errors.New("get user error")
	ErrGetQuestion         = errors.New("get question error")
	ErrSaveAnswer          = errors.New("save answer error")
)

func decodeGRPCError(err, notFound error) error {
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
			mapped = ErrInvalidRequestBody
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
