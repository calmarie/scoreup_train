package users

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	CreateUser(ctx context.Context) (User, error)
	GetUserByTelegramID(ctx context.Context, telegramID int64) (User, error)
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// hardcoded user creation, in advance it will be supplemented with telegram user ID
func (r *Repository) CreateUser(ctx context.Context) (User, error) {
	const create = `
		INSERT INTO users DEFAULT VALUES
		RETURNING id, telegram_id, created_at;`

	var user User
	if err := r.db.QueryRow(ctx, create).Scan(
		&user.Id,
		&user.Telegram_id,
		&user.Created_at,
	); err != nil {
		return User{}, createUserError(err)
	}

	return user, nil
}

func (r *Repository) GetUserByTelegramID(ctx context.Context, telegramID int64) (User, error) {
	const getUser = `
		SELECT id, telegram_id, created_at
		FROM users
		WHERE telegram_id = $1;`

	var user User
	if err := r.db.QueryRow(ctx, getUser, telegramID).Scan(
		&user.Id,
		&user.Telegram_id,
		&user.Created_at,
	); err != nil {
		return User{}, getUserError(err)
	}

	return user, nil
}
