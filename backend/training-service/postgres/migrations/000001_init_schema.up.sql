CREATE TABLE questions (
    id SERIAL PRIMARY KEY,
    type VARCHAR(30) NOT NULL CHECK (
        type IN ('radio', 'checkbox', 'text', 'code')
    ),
    difficulty INTEGER NOT NULL CHECK (difficulty BETWEEN 1 AND 10),
    topic VARCHAR(100) NOT NULL,
    skill VARCHAR(150),
    text TEXT NOT NULL,
    image_url TEXT,
    options JSONB,
    correct_answer JSONB NOT NULL,
    explanation TEXT,
    data JSONB,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    xp INTEGER NOT NULL DEFAULT 100
);

CREATE TABLE user_answers (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    question_id INTEGER NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    answer JSONB NOT NULL,
    is_correct BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_answers_user_question
ON user_answers(user_id, question_id);

CREATE TABLE user_solved_questions (
    user_id BIGINT NOT NULL,
    question_id INTEGER NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    solved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, question_id)
);
