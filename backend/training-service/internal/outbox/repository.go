package outbox

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OutboxRepository interface {
	GetEvents(ctx context.Context, limit int) ([]Event, error)
	MarkPublished(ctx context.Context, id int64) error
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func SaveEvent(ctx context.Context, tx pgx.Tx, event Event) (int64, error) {
	var eventID int64
	err := tx.QueryRow(ctx, `
		INSERT INTO outbox_events (topic, event_key, payload)
		VALUES ($1, $2, $3)
		RETURNING id`, event.Topic, event.EventKey, event.Payload,
	).Scan(&eventID)
	if err != nil {
		return 0, fmt.Errorf("save outbox event: %w", err)
	}
	return eventID, nil
}

func (r *Repository) GetEvents(ctx context.Context, limit int) ([]Event, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, topic, event_key, payload, published_at
		FROM outbox_events
		WHERE published_at IS NULL
		ORDER BY id
		LIMIT $1`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get pending outbox events: %w", err)
	}

	events, err := pgx.CollectRows(rows, pgx.RowToStructByName[Event])
	if err != nil {
		return nil, fmt.Errorf("read pending outbox events: %w", err)
	}
	return events, nil
}

func (r *Repository) MarkPublished(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE outbox_events
		SET published_at = NOW()
		WHERE id = $1 AND published_at IS NULL`, id,
	)
	if err != nil {
		return fmt.Errorf("mark outbox event %d published: %w", id, err)
	}
	return nil
}
