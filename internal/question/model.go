package question

import (
	"encoding/json"
	"time"
)

type Question struct {
	ID            int             `json:"id"`
	Type          string          `json:"type"`
	Difficulty    int             `json:"difficulty"`
	Topic         string          `json:"topic"`
	Skill         *string         `json:"skill"`
	Text          string          `json:"text"`
	ImageURL      *string         `json:"image_url"`
	Options       json.RawMessage `json:"options"`
	CorrectAnswer json.RawMessage `json:"correct_answer"`
	Explanation   *string         `json:"explanation"`
	Data          json.RawMessage `json:"data"`
	IsActive      bool            `json:"is_active"`
}

type Answer struct {
	ID          int             `json:"id"`
	User_ID     int             `json:"user_id"`
	Question_ID int             `json:"question_id"`
	Answer      json.RawMessage `json:"answer"`
	Is_correct  bool            `json:"is_correct"`
	Created_at  time.Time       `json:"created_at"`
}

type CodeAnswer struct {
	Language string `json:"language"`
	Code     string `json:"code"`
}
