-- Puzzle rush sessions table
CREATE TABLE IF NOT EXISTS puzzle_rush_sessions (
    id INTEGER PRIMARY KEY,
    profile_id INTEGER NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    difficulty TEXT NOT NULL CHECK(difficulty IN ('easy', 'medium', 'hard')),
    score INTEGER DEFAULT 0,
    mistakes_made INTEGER DEFAULT 0,
    mistakes_allowed INTEGER NOT NULL CHECK(mistakes_allowed IN (1, 3, 5)),
    total_time_seconds REAL DEFAULT 0,
    completed_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Puzzle rush attempts table - tracks each flashcard within a session
CREATE TABLE IF NOT EXISTS puzzle_rush_attempts (
    id INTEGER PRIMARY KEY,
    session_id INTEGER NOT NULL REFERENCES puzzle_rush_sessions(id) ON DELETE CASCADE,
    flashcard_id INTEGER NOT NULL REFERENCES flashcards(id) ON DELETE CASCADE,
    was_correct BOOLEAN NOT NULL,
    time_seconds REAL DEFAULT 0,
    attempt_number INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_puzzle_rush_sessions_profile ON puzzle_rush_sessions(profile_id);
CREATE INDEX IF NOT EXISTS idx_puzzle_rush_sessions_created ON puzzle_rush_sessions(created_at);
CREATE INDEX IF NOT EXISTS idx_puzzle_rush_sessions_completed ON puzzle_rush_sessions(completed_at);
CREATE INDEX IF NOT EXISTS idx_puzzle_rush_attempts_session ON puzzle_rush_attempts(session_id);
CREATE INDEX IF NOT EXISTS idx_puzzle_rush_attempts_flashcard ON puzzle_rush_attempts(flashcard_id);
