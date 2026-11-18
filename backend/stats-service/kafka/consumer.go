package kafka

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/calmarie/scoreup_train/backend/stats-service/internal/stats"
	"github.com/twmb/franz-go/pkg/kgo"
)

type QuestionAttemptEvent = stats.QuestionAttemptEvent

const maxBatchSize = 32

// Consume updates user stats until the context is canceled or an error occurs.
func Consume(ctx context.Context, logger *slog.Logger, service stats.StatsService) error {
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(os.Getenv("KAFKA_URL")),
		kgo.ConsumerGroup("stats-service"),
		kgo.ConsumeTopics("question-solved"),
		kgo.DisableAutoCommit(),
		kgo.BlockRebalanceOnPoll(),
	)
	if err != nil {
		return fmt.Errorf("create Kafka consumer: %w", err)
	}
	defer cl.CloseAllowingRebalance()

	for {
		fetches := cl.PollRecords(ctx, maxBatchSize)
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("consume Kafka event: %w", err)
		}

		if errs := fetches.Errors(); len(errs) > 0 {
			return fmt.Errorf("fetch Kafka topic %s partition %d: %w", errs[0].Topic, errs[0].Partition, errs[0].Err)
		}
		if err := processBatch(ctx, logger, service, fetches.Records(), cl.CommitRecords); err != nil {
			return err
		}
		cl.AllowRebalance()
	}
}
