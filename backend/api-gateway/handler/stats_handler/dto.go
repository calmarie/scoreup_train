package stats_handler

type Stats struct {
	UserID               int64   `json:"user_id"`
	XP                   int64   `json:"xp"`
	AccuracyPercent      float64 `json:"accuracy_percent"`
	UniqueSolved         int64   `json:"unique_solved"`
	ActiveDays           int32   `json:"active_days"`
	CurrentStreak        int64   `json:"current_streak"`
	MaxStreak            int64   `json:"max_streak"`
	CurrentCorrectStreak int64   `json:"current_correct_streak"`
	BestCorrectStreak    int64   `json:"best_correct_streak"`
	LastSolvedOn         *string `json:"last_solved_on"`
}

type XP struct {
	XP int64 `json:"xp"`
}

type CorrectStreak struct {
	CurrentCorrectStreak int64 `json:"current_correct_streak"`
}

type Streak struct {
	CurrentStreak int64 `json:"current_streak"`
}
