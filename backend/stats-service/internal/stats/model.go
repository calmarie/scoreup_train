package stats

import "time"

// QuestionAttemptEvent is the attempt published by the training service.
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

type UserStats struct {
	XP                   int64
	TotalAnswers         int64
	CorrectAnswers       int64
	UniqueSolved         int64
	ActiveDays           int
	CurrentStreak        int
	MaxStreak            int
	CurrentCorrectStreak int
	BestCorrectStreak    int
	LastSolvedOn         *time.Time
}
