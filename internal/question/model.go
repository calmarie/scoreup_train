package question

import "encoding/json"

type Question struct {
	ID            int             `json:"id"`
	Type          string          `json:"type"`
	Difficulty    int             `json:"difficulty"`
	Topic         string          `json:"topic"`
	Skill         *string         `json:"skill"`
	Text          string          `json:"text"`
	ImageURL      *string         `json:"image_url"`
	Options       json.RawMessage `json:"options"`
	CorrectAnswer string          `json:"correct_answer"`
	Explanation   *string         `json:"explanation"`
	Data          json.RawMessage `json:"data"`
	IsActive      bool            `json:"is_active"`
}

type Answer struct {
	Text string `json:"text"`
}
