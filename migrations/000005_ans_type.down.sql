ALTER TABLE questions 
ALTER COLUMN correct_answer TYPE TEXT 
USING to_text(correct_answer);