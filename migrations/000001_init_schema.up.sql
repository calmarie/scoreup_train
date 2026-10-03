CREATE TABLE IF NOT EXISTS questions (
    id SERIAL PRIMARY KEY,

    type VARCHAR(30) NOT NULL CHECK (
        type IN (
            'radio',
            'checkbox',
            'text',
            'code'
        )
    ),

    difficulty INTEGER NOT NULL CHECK (
        difficulty BETWEEN 1 AND 10
    ),

    topic VARCHAR(100) NOT NULL,
    skill VARCHAR(150),
    text TEXT NOT NULL,
    image_url TEXT,
    options JSONB,
    correct_answer TEXT NOT NULL,
    explanation TEXT,
    data JSONB,
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);
