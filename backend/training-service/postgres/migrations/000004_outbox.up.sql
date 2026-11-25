-- migrate create -ext sql -dir backend/training-service/postgres/migrations -seq outbox

CREATE TABLE outbox_events (
    id BIGSERIAL PRIMARY KEY,
    topic TEXT NOT NULL,
    event_key TEXT NOT NULL,
    payload JSONB NOT NULL,
    published_at TIMESTAMPTZ
);

CREATE INDEX idx_outbox_pending
ON outbox_events (id)
WHERE published_at IS NULL;