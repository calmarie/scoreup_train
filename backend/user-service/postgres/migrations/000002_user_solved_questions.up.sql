CREATE TABLE user_solved_questions (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- The training service owns questions in a separate database.
    question_id INTEGER NOT NULL,
    solved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, question_id)
);
