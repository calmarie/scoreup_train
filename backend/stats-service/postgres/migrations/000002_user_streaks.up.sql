ALTER TABLE user_stats
    ADD COLUMN last_solved_on DATE,
    ADD COLUMN current_correct_streak INTEGER NOT NULL DEFAULT 0;
