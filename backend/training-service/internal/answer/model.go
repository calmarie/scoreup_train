package answer

import (
	"encoding/json"
	"time"
)

type SubmitAnswerRequest struct {
	UserID     int64
	QuestionID int
	Answer     json.RawMessage
}

type Answer struct {
	ID         int64
	UserID     int64
	QuestionID int
	Topic      string
	Difficulty int
	Answer     json.RawMessage
	IsCorrect  bool
	FirstSolve bool
	CreatedAt  time.Time
	XP         int
}

const AttemptTopic = "question-solved"

type QuestionAttemptEvent struct {
	AnswerID   int64     `json:"answer_id"`
	UserID     int64     `json:"user_id"`
	QuestionID int64     `json:"question_id"`
	Topic      string    `json:"topic"`
	Difficulty int       `json:"difficulty"`
	IsCorrect  bool      `json:"is_correct"`
	XP         int64     `json:"xp"`
	OccurredAt time.Time `json:"occurred_at"`
	FirstSolve bool      `json:"first_solve"`
}

// AttemptEvent awards the question's XP only on the user's first solve.
func (a Answer) AttemptEvent() QuestionAttemptEvent {
	var awardedXP int64
	if a.FirstSolve {
		awardedXP = int64(a.XP)
	}
	return QuestionAttemptEvent{
		AnswerID:   a.ID,
		UserID:     a.UserID,
		QuestionID: int64(a.QuestionID),
		Topic:      a.Topic,
		Difficulty: a.Difficulty,
		IsCorrect:  a.IsCorrect,
		XP:         awardedXP,
		OccurredAt: a.CreatedAt,
		FirstSolve: a.FirstSolve,
	}
}

type CodeAnswer struct {
	Language string
	Code     string
}
