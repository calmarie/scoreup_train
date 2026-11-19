package user_handler

import "time"

type User struct {
	ID         int64     `json:"id"`
	TelegramID int64     `json:"telegram_id"`
	CreatedAt  time.Time `json:"created_at"`
}
