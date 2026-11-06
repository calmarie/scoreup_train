ALTER TABLE questions
ALTER COLUMN correct_answer TYPE JSONB
USING to_jsonb(correct_answer);
