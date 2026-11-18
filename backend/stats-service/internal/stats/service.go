package stats

import (
	"context"
	"time"
)

type StatsService interface {
	Save(ctx context.Context, event QuestionAttemptEvent) (bool, error)
}

type Service struct {
	repository StatsRepository
}

func NewService(repository StatsRepository) *Service {
	return &Service{repository: repository}
}

// Save updates the user's totals and returns false if the attempt was already counted.
func (s *Service) Save(ctx context.Context, event QuestionAttemptEvent) (bool, error) {
	return s.repository.Update(ctx, event.UserID, event.AnswerID, func(totals *UserStats) {

		totals.TotalAnswers++

		totals.XP += event.XP

		if event.IsCorrect {
			totals.CorrectAnswers++
		}
		if event.FirstSolve {
			totals.UniqueSolved++
		}

		day := event.OccurredAt.UTC().Truncate(24 * time.Hour)
		active := newActiveDay(totals.LastSolvedOn, day, event.IsCorrect)
		totals.CurrentStreak = dailyStreak(totals.CurrentStreak, totals.LastSolvedOn, day, active)
		totals.MaxStreak = max(totals.MaxStreak, totals.CurrentStreak)
		if active {
			totals.ActiveDays++
			totals.LastSolvedOn = &day
		}

		totals.CurrentCorrectStreak = correctStreak(totals.CurrentCorrectStreak, event.IsCorrect)
		totals.BestCorrectStreak = max(totals.BestCorrectStreak, totals.CurrentCorrectStreak)
	})
}

func newActiveDay(lastSolved *time.Time, day time.Time, correct bool) bool {
	return correct && (lastSolved == nil || day.After(*lastSolved))
}

func dailyStreak(current int, lastSolved *time.Time, day time.Time, active bool) int {
	if lastSolved != nil && day.After(lastSolved.AddDate(0, 0, 1)) {
		current = 0
	}
	if active {
		current++
	}
	return current
}

func correctStreak(current int, correct bool) int {
	if correct {
		return current + 1
	}
	return 0
}
