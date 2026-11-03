package users

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrCreateUser          = errors.New("create user error")
	ErrUserAlreadyExists   = errors.New("user already exists")
	ErrDatabaseUnavailable = errors.New("database temporarily unavailable")
)

func createUserError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("create user: %w", err)
	}

	var postgresErr *pgconn.PgError
	if errors.As(err, &postgresErr) {
		if postgresErr.Code == "23505" {
			return fmt.Errorf("create user: %w: %w", ErrUserAlreadyExists, err)
		}
		if postgresUnavailable(postgresErr.Code) {
			return fmt.Errorf("create user: %w: %w", ErrDatabaseUnavailable, err)
		}
	}

	var networkErr net.Error
	if errors.As(err, &networkErr) {
		return fmt.Errorf("create user: %w: %w", ErrDatabaseUnavailable, err)
	}

	return fmt.Errorf("create user: %w: %w", ErrCreateUser, err)
}

func postgresUnavailable(code string) bool {
	return len(code) >= 2 && (code[:2] == "08" || code[:2] == "53") ||
		code == "57P01" || code == "57P02" || code == "57P03"
}
