package training_handler

import (
	"encoding/json"
	"time"
)

type Question struct {
	ID          int             `json:"id"`
	Type        string          `json:"type"`
	Difficulty  int             `json:"difficulty"`
	Topic       string          `json:"topic"`
	Skill       *string         `json:"skill"`
	Text        string          `json:"text"`
	ImageURL    *string         `json:"image_url"`
	Options     json.RawMessage `json:"options"`
	Explanation *string         `json:"explanation"`
	Data        json.RawMessage `json:"data"`
	IsActive    bool            `json:"is_active"`
}

type SubmitAnswerRequest struct {
	TelegramID int64           `json:"telegram_id"`
	QuestionID int32           `json:"question_id"`
	Answer     json.RawMessage `json:"answer"`
}

type Answer struct {
	ID        int64     `json:"id"`
	IsCorrect bool      `json:"is_correct"`
	CreatedAt time.Time `json:"created_at"`
}

type CodeAnswer struct {
	Language string `json:"language"`
	Code     string `json:"code"`
}
