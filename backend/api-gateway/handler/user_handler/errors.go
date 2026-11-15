package user_handler

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
	ErrUserNotFound        = errors.New("user not found")
	ErrUserAlreadyExists   = errors.New("user already exists")
	ErrServiceUnavailable  = errors.New("user service unavailable")
	ErrCreateUser          = errors.New("create user error")
	ErrGetUser             = errors.New("get user error")
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
			mapped = ErrInvalidTelegramID
		case codes.NotFound:
			mapped = ErrUserNotFound
		case codes.AlreadyExists:
			mapped = ErrUserAlreadyExists
		case codes.Unavailable:
			mapped = ErrServiceUnavailable
		}
	}
	if mapped == nil {
		return err
	}
	return fmt.Errorf("%w: %w", mapped, err)
}
