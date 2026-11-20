package daily_stats

import (
	"context"
	"math"
)

type DailyStatsService interface {
	Save(ctx context.Context, event QuestionAttemptEvent) (bool, error)
}

type Service struct {
	repository StatsRepository
}

func NewService(repository StatsRepository) *Service {
	return &Service{repository: repository}
}

func (s *Service) GetDailyStats(ctx context.Context, userID int64) (UserStatsDTO, error) {
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

	})
}
