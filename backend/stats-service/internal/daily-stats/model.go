package daily_stats

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

type UserStatsDTO struct {
	UserID               int64      `json:"user_id"`
	XP                   int64      `json:"xp"`
	TotalAnswers         int64      `json:"total_answers"`
	CorrectAnswers       int64      `json:"correct_answers"`
	AccuracyPercent      float64    `json:"accuracy_percent"`
	UniqueSolved         int64      `json:"unique_solved"`
	ActiveDays           int        `json:"active_days"`
	CurrentStreak        int        `json:"current_streak"`
	MaxStreak            int        `json:"max_streak"`
	CurrentCorrectStreak int        `json:"current_correct_streak"`
	BestCorrectStreak    int        `json:"best_correct_streak"`
	LastSolvedOn         *time.Time `json:"last_solved_on"`
}
