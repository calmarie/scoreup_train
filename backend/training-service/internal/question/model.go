package question

import (
	"encoding/json"
	"time"
)

type Question struct {
	ID          int
	Type        string
	Difficulty  int
	Topic       string
	Skill       *string
	Text        string
	ImageURL    *string
	Options     json.RawMessage
	Explanation *string
	Data        json.RawMessage
	IsActive    bool
	XP          int
}

type SubmitAnswerRequest struct {
	UserID     int64
	QuestionID int
	Answer     json.RawMessage
}

type Answer struct {
	ID         int64
	UserID     int64
	QuestionID int
	Answer     json.RawMessage
	IsCorrect  bool
	CreatedAt  time.Time
	XP         int
}

type CodeAnswer struct {
	Language string
	Code     string
}
