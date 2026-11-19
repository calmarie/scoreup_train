package stats

import (
	"context"
	"testing"
	"time"
)

type statsReadRepository struct {
	StatsRepository
	totals UserStatsDTO
}

func (r statsReadRepository) GetStats(context.Context, int64) (UserStatsDTO, error) {
	return r.totals, nil
}

func TestGetStatsCalculatesAccuracy(t *testing.T) {
	for _, tc := range []struct {
		name   string
		totals UserStatsDTO
		want   float64
	}{
		{"fraction", UserStatsDTO{TotalAnswers: 7, CorrectAnswers: 4}, 57.14},
		{"no attempts", UserStatsDTO{}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := NewService(statsReadRepository{totals: tc.totals})
			got, err := service.GetStats(t.Context(), 42)
			if err != nil || got.AccuracyPercent != tc.want {
				t.Fatalf("accuracy = %g, err = %v, want %g", got.AccuracyPercent, err, tc.want)
			}
		})
	}
}

func TestDailyStreak(t *testing.T) {
	today := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	yesterday := today.AddDate(0, 0, -1)
	twoDaysAgo := today.AddDate(0, 0, -2)
	for _, tc := range []struct {
		name       string
		lastSolved *time.Time
		current    int
		correct    bool
		active     bool
		streak     int
	}{
		{"first miss", nil, 0, false, false, 0},
		{"first solve", nil, 0, true, true, 1},
		{"same day", &today, 3, true, false, 3},
		{"miss the next day", &yesterday, 3, false, false, 3},
		{"solve the next day", &yesterday, 3, true, true, 4},
		{"miss after a gap", &twoDaysAgo, 3, false, false, 0},
		{"solve after a gap", &twoDaysAgo, 3, true, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			active := newActiveDay(tc.lastSolved, today, tc.correct)
			streak := dailyStreak(tc.current, tc.lastSolved, today, active)
			if active != tc.active || streak != tc.streak {
				t.Fatalf("active day = %t, streak = %d, want %t and %d", active, streak, tc.active, tc.streak)
			}
		})
	}
}
