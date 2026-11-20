package users

import (
	"time"

	"github.com/calmarie/scoreup_train/backend/user-service/internal/auth"
)

type User struct {
	ID         int64
	TelegramID int64
	CreatedAt  time.Time
}

type CreateUserResult struct {
	User    User
	Session auth.Session
	Created bool
}
