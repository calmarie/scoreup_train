package answer

import (
	"testing"
	"time"
)

func TestAttemptEventAwardsXPOnlyForFirstSolve(t *testing.T) {
	createdAt := time.Date(2026, time.November, 14, 12, 30, 45, 0, time.UTC)
	answer := Answer{
		ID: 19, UserID: 7, QuestionID: 3, Topic: "math", Difficulty: 3,
		XP: 75, IsCorrect: true, FirstSolve: true, CreatedAt: createdAt,
	}
	want := QuestionAttemptEvent{
		AnswerID: 19, UserID: 7, QuestionID: 3, Topic: "math", Difficulty: 3,
		XP: 75, IsCorrect: true, FirstSolve: true, OccurredAt: createdAt,
	}
	if event := answer.AttemptEvent(); event != want {
		t.Fatalf("first solve event = %+v, want %+v", event, want)
	}

	answer.FirstSolve = false
	if event := answer.AttemptEvent(); event.XP != 0 || event.FirstSolve || !event.IsCorrect {
		t.Fatalf("repeated correct attempt should have zero XP: %+v", event)
	}

	answer.IsCorrect = false
	if event := answer.AttemptEvent(); event.XP != 0 || event.FirstSolve || event.IsCorrect {
		t.Fatalf("incorrect attempt should have zero XP: %+v", event)
	}
}
