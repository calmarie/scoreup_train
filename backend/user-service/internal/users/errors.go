package users

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrCreateUser          = errors.New("create user error")
	ErrGetUser             = errors.New("get user error")
	ErrUserAlreadyExists   = errors.New("user already exists")
	ErrUserNotFound        = errors.New("user not found")
	ErrInvalidTelegramID   = errors.New("invalid telegram id")
	ErrDatabaseUnavailable = errors.New("database temporarily unavailable")
)

func createUserError(err error) error {
	var postgresErr *pgconn.PgError
	if errors.As(err, &postgresErr) {
		if postgresErr.Code == "23505" {
			return fmt.Errorf("create user: %w: %w", ErrUserAlreadyExists, err)
		}
	}
	return databaseError("create user", ErrCreateUser, err)
}

func getUserError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("get user by telegram id: %w", ErrUserNotFound)
	}

	return databaseError("get user by telegram id", ErrGetUser, err)
}

func databaseError(operation string, fallback, err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%s: %w", operation, err)
	}
	var postgresErr *pgconn.PgError
	var networkErr net.Error
	if (errors.As(err, &postgresErr) && postgresUnavailable(postgresErr.Code)) || errors.As(err, &networkErr) {
		return fmt.Errorf("%s: %w: %w", operation, ErrDatabaseUnavailable, err)
	}
	return fmt.Errorf("%s: %w: %w", operation, fallback, err)
}

// postgresUnavailable recognizes SQLSTATE codes for connection failures, resource shortages, and server shutdown or startup.
func postgresUnavailable(code string) bool {
	return len(code) >= 2 && (code[:2] == "08" || code[:2] == "53") ||
		code == "57P01" || code == "57P02" || code == "57P03"
}
