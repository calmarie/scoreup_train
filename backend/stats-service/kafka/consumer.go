package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"
)

// QuestionAttemptEvent is the JSON payload read from the training service's question-solved topic.
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

// Consume records attempt IDs until the context is canceled or an error occurs.
func Consume(ctx context.Context, logger *slog.Logger, db *pgxpool.Pool) error {
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(os.Getenv("KAFKA_URL")),
		kgo.ConsumerGroup("stats-service"),
		kgo.ConsumeTopics("question-solved"),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		return fmt.Errorf("create Kafka consumer: %w", err)
	}
	defer cl.Close()

	for {
		fetches := cl.PollRecords(ctx, 1)
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("consume Kafka event: %w", err)
		}

		if errs := fetches.Errors(); len(errs) > 0 {
			return fmt.Errorf("fetch Kafka topic %s partition %d: %w", errs[0].Topic, errs[0].Partition, errs[0].Err)
		}
		for _, record := range fetches.Records() {
			var event QuestionAttemptEvent
			if err := json.Unmarshal(record.Value, &event); err != nil {
				return fmt.Errorf("decode Kafka event at %s/%d/%d: %w", record.Topic, record.Partition, record.Offset, err)
			}

			// The answer ID stays the same when Kafka redelivers an attempt.
			result, err := db.Exec(ctx,
				`INSERT INTO processed_events (event_id) VALUES ($1) ON CONFLICT (event_id) DO NOTHING`,
				event.AnswerID,
			)
			if err != nil {
				return fmt.Errorf("save processed event: %w", err)
			}
			// Commit after the database write, including when the ID was already recorded.
			if err := cl.CommitRecords(ctx, record); err != nil {
				return fmt.Errorf("commit Kafka event: %w", err)
			}
			if result.RowsAffected() == 0 {
				continue
			}
			logger.Info("question attempt received",
				"answer_id", event.AnswerID,
				"user_id", event.UserID,
				"question_id", event.QuestionID,
				"topic", event.Topic,
				"difficulty", event.Difficulty,
				"is_correct", event.IsCorrect,
				"xp", event.XP,
				"occurred_at", event.OccurredAt,
				"first_solve", event.FirstSolve,
			)
		}
	}
}
