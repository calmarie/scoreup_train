-- The consumer uses the training answer ID to recognize redelivered Kafka events.
CREATE TABLE processed_events (
    event_id BIGINT PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_stats (
    user_id BIGINT PRIMARY KEY,
    xp BIGINT NOT NULL DEFAULT 0,
    current_streak INTEGER NOT NULL DEFAULT 0,
    max_streak INTEGER NOT NULL DEFAULT 0,
    total_answers BIGINT NOT NULL DEFAULT 0,
    correct_answers BIGINT NOT NULL DEFAULT 0,
    unique_solved BIGINT NOT NULL DEFAULT 0,
    active_days INTEGER NOT NULL DEFAULT 0,
    best_correct_streak INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE user_topic_stats (
    user_id BIGINT NOT NULL,
    topic VARCHAR(100) NOT NULL,
    attempts BIGINT NOT NULL DEFAULT 0,
    correct BIGINT NOT NULL DEFAULT 0,
    solved BIGINT NOT NULL DEFAULT 0,
    xp BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, topic)
);

CREATE TABLE user_daily_stats (
    user_id BIGINT NOT NULL,
    date DATE NOT NULL,
    attempts BIGINT NOT NULL DEFAULT 0,
    correct BIGINT NOT NULL DEFAULT 0,
    solved BIGINT NOT NULL DEFAULT 0,
    xp BIGINT NOT NULL DEFAULT 0,
    active_seconds BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, date)
);
