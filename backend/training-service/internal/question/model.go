package question

import "encoding/json"

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
