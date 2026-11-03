--to create new pair of migration files: migrate create -ext sql -dir migrations -seq 'name'

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    telegram_id BIGINT UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);