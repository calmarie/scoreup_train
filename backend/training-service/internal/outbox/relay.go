package outbox

import (
	"context"
	"log/slog"
	"time"

	"github.com/calmarie/scoreup_train/backend/training-service/kafka"
	"github.com/twmb/franz-go/pkg/kgo"
)

type OutboxRelay interface {
}

type Relay struct {
	repository OutboxRepository
}

func NewRelay(repository OutboxRepository) *Relay {
	return &Relay{repository: repository}
}

func (rel *Relay) Relay(ctx context.Context, cl *kgo.Client, logger *slog.Logger) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := rel.processPending(ctx, cl); err != nil && ctx.Err() == nil {
				logger.ErrorContext(ctx, "process pending outbox events", "error", err)
			}
		}
	}
}

func (rel *Relay) processPending(ctx context.Context, cl *kgo.Client) error {
	events, err := rel.repository.GetEvents(ctx, 50)
	if err != nil {
		return err
	}
	for _, event := range events {
		//kgo.Record is creating here to escape import cycle
		record := &kgo.Record{Topic: event.Topic, Key: []byte(event.EventKey), Value: event.Payload}
		if err := kafka.Produce(ctx, cl, record); err != nil {
			return err
		}
		if err := rel.repository.MarkPublished(ctx, event.ID); err != nil {
			return err
		}
	}

	return nil
}
