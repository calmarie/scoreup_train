package users

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	CreateUser(ctx context.Context, telegramID int64) (User, error)
	GetUserByTelegramID(ctx context.Context, telegramID int64) (User, error)
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// CreateUser stores the Telegram ID and returns the user with its database defaults.
func (r *Repository) CreateUser(ctx context.Context, telegramID int64) (User, error) {
	const create = `
		INSERT INTO users (telegram_id) VALUES ($1)
		RETURNING id, telegram_id, created_at, xp_total;`

	var user User
	if err := r.db.QueryRow(ctx, create, telegramID).Scan(
		&user.ID,
		&user.TelegramID,
		&user.CreatedAt,
		&user.XPTotal,
	); err != nil {
		return User{}, createUserError(err)
	}

	return user, nil
}

func (r *Repository) GetUserByTelegramID(ctx context.Context, telegramID int64) (User, error) {
	const getUser = `
		SELECT id, telegram_id, created_at, xp_total
		FROM users
		WHERE telegram_id = $1;`

	var user User
	if err := r.db.QueryRow(ctx, getUser, telegramID).Scan(
		&user.ID,
		&user.TelegramID,
		&user.CreatedAt,
		&user.XPTotal,
	); err != nil {
		return User{}, getUserError(err)
	}

	return user, nil
}
