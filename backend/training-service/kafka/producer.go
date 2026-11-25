package kafka

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

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

// Produce sends a record and waits for Kafka to acknowledge it.
func Produce(ctx context.Context, cl *kgo.Client, record *kgo.Record) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := cl.ProduceSync(ctx, record).FirstErr(); err != nil {
		return fmt.Errorf("produce Kafka event: %w", err)
	}
	return nil
}
