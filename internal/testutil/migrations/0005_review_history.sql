-- Review history table for tracking flashcard reviews with timing data
CREATE TABLE IF NOT EXISTS review_history (
    id INTEGER PRIMARY KEY,
    flashcard_id INTEGER NOT NULL REFERENCES flashcards(id) ON DELETE CASCADE,
    quality INTEGER NOT NULL, -- 0=Again, 1=Hard, 2=Good, 3=Easy
    time_seconds REAL NOT NULL, -- Time taken to answer in seconds
    reviewed_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_review_history_flashcard ON review_history(flashcard_id);
CREATE INDEX IF NOT EXISTS idx_review_history_reviewed_at ON review_history(reviewed_at);
