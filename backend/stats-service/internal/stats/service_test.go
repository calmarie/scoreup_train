package stats

import (
	"testing"
	"time"
)

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
