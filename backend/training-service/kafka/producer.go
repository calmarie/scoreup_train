package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/calmarie/scoreup_train/backend/training-service/internal/question"
	"github.com/twmb/franz-go/pkg/kgo"
)

// QuestionAttemptEvent is the JSON payload expected by the stats service's Kafka consumer.
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

// NewProducerClient creates a Kafka client from KAFKA_URL. The caller closes it.
func NewProducerClient() (*kgo.Client, error) {
	broker := os.Getenv("KAFKA_URL")
	if broker == "" {
		return nil, fmt.Errorf("environment variable KAFKA_URL is required")
	}

	cl, err := kgo.NewClient(
		kgo.SeedBrokers(broker),
	)
	if err != nil {
		return nil, fmt.Errorf("create Kafka producer: %w", err)
	}
	return cl, nil
}

// Produce publishes one attempt to question-solved and waits for Kafka to acknowledge it.
func Produce(ctx context.Context, logger *slog.Logger, cl *kgo.Client, answer question.Answer) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	event := newAttemptEvent(answer)
	value, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode Kafka event: %w", err)
	}

	// Key attempts by user so Kafka can route them to the same partition.
	record := &kgo.Record{
		Topic: "question-solved",
		Key:   []byte(strconv.FormatInt(event.UserID, 10)),
		Value: value,
	}
	if err := cl.ProduceSync(ctx, record).FirstErr(); err != nil {
		return fmt.Errorf("produce Kafka event: %w", err)
	}

	logger.Info("question attempt sent", "answer_id", event.AnswerID, "user_id", event.UserID)
	return nil
}

// newAttemptEvent includes the question's XP only on the user's first solve.
func newAttemptEvent(answer question.Answer) QuestionAttemptEvent {
	var awardedXP int64
	if answer.FirstSolve {
		awardedXP = int64(answer.XP)
	}
	return QuestionAttemptEvent{
		AnswerID:   answer.ID,
		UserID:     answer.UserID,
		QuestionID: int64(answer.QuestionID),
		Topic:      answer.Topic,
		Difficulty: answer.Difficulty,
		IsCorrect:  answer.IsCorrect,
		XP:         awardedXP,
		OccurredAt: answer.CreatedAt,
		FirstSolve: answer.FirstSolve,
	}
}
