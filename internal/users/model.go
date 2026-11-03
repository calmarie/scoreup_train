package users

import "time"

type User struct {
	Id          int64     `json:"id"`
	Telegram_id *int64    `json:"telegram_id"`
	Created_at  time.Time `json:"created_at"`
}
