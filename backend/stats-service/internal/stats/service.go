package stats

import (
	"context"
	"math"
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

func (s *Service) GetStats(ctx context.Context, userID int64) (UserStatsDTO, error) {
	totals, err := s.repository.GetStats(ctx, userID)
	if err != nil {
		return totals, err
	}
	if totals.TotalAnswers > 0 {
		percent := float64(totals.CorrectAnswers) / float64(totals.TotalAnswers) * 100
		totals.AccuracyPercent = math.Round(percent*100) / 100
	}
	return totals, nil
}

func (s *Service) GetXP(ctx context.Context, userID int64) (int64, error) {
	return s.repository.GetXP(ctx, userID)
}

func (s *Service) GetCurrentStreak(ctx context.Context, userID int64) (int64, error) {
	return s.repository.GetCurrentStreak(ctx, userID)
}

func (s *Service) GetCurrentCorrectStreak(ctx context.Context, userID int64) (int64, error) {
	return s.repository.GetCurrentCorrectStreak(ctx, userID)
}

func (s *Service) GetDailyStats(ctx context.Context, userID int64, day time.Time) (UserDailyStats, error) {
	return s.repository.GetDailyStats(ctx, userID, eventDay(day))
}

func (s *Service) GetWeeklyStats(ctx context.Context, userID int64) ([]UserDailyStats, error) {
	today := eventDay(time.Now())
	return s.repository.GetDailyStatsRange(ctx, userID, today.AddDate(0, 0, -6), today)
}

func (s *Service) GetTopicStats(ctx context.Context, userID int64, topic string) (UserTopicStats, error) {
	return s.repository.GetTopicStats(ctx, userID, topic)
}

func (s *Service) GetAllTopicStats(ctx context.Context, userID int64) ([]UserTopicStats, error) {
	return s.repository.GetAllTopicStats(ctx, userID)
}

// Save updates the user's stats and returns false if the attempt was already counted.
func (s *Service) Save(ctx context.Context, event QuestionAttemptEvent) (bool, error) {
	day := eventDay(event.OccurredAt)
	return s.repository.Update(ctx, event, day, func(totals *UserStats) {

		totals.TotalAnswers++

		totals.XP += event.XP

		if event.IsCorrect {
			totals.CorrectAnswers++
		}
		if event.FirstSolve {
			totals.UniqueSolved++
		}

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

func eventDay(occurredAt time.Time) time.Time {
	return occurredAt.UTC().Truncate(24 * time.Hour)
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
