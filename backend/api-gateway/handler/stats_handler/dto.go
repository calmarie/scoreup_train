package stats_handler

import (
	"fmt"

	statspb "github.com/calmarie/scoreup_train/backend/gen/stats"
)

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

type DailyStats struct {
	Date     string `json:"date"`
	Attempts int64  `json:"attempts"`
	Correct  int64  `json:"correct"`
	Solved   int64  `json:"solved"`
	XP       int64  `json:"xp"`
}

type WeeklyStats struct {
	Days []DailyStats `json:"days"`
}

type TopicStats struct {
	Topic    string `json:"topic"`
	Attempts int64  `json:"attempts"`
	Correct  int64  `json:"correct"`
	Solved   int64  `json:"solved"`
	XP       int64  `json:"xp"`
}

type AllTopicStats struct {
	Topics []TopicStats `json:"topics"`
}

func dailyStatsJSON(reply *statspb.DailyStats) DailyStats {
	day := reply.GetDate()
	return DailyStats{
		Date:     fmt.Sprintf("%04d-%02d-%02d", day.GetYear(), day.GetMonth(), day.GetDay()),
		Attempts: reply.GetAttempts(),
		Correct:  reply.GetCorrect(),
		Solved:   reply.GetSolved(),
		XP:       reply.GetXp(),
	}
}

func topicStatsJSON(reply *statspb.TopicStats) TopicStats {
	return TopicStats{
		Topic:    reply.GetTopic(),
		Attempts: reply.GetAttempts(),
		Correct:  reply.GetCorrect(),
		Solved:   reply.GetSolved(),
		XP:       reply.GetXp(),
	}
}
