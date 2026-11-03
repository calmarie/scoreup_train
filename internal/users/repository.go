package users

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	CreateUser(ctx context.Context) (User, error)
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
