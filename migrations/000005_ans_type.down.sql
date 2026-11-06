ALTER TABLE questions 
ALTER COLUMN correct_answer TYPE TEXT 
USING correct_answer::text;