-- Users/profiles being analyzed
CREATE TABLE IF NOT EXISTS profiles (
    id INTEGER PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_sync_at DATETIME
);

-- Imported games
CREATE TABLE IF NOT EXISTS games (
    id INTEGER PRIMARY KEY,
    profile_id INTEGER NOT NULL REFERENCES profiles(id),
    chess_com_id TEXT UNIQUE NOT NULL,
    pgn TEXT NOT NULL,
    time_class TEXT NOT NULL, -- bullet, blitz, rapid, daily
    result TEXT NOT NULL, -- win, loss, draw
    played_as TEXT NOT NULL, -- white, black
    opponent TEXT NOT NULL,
    played_at DATETIME NOT NULL,
    eco_code TEXT, -- ECO code e.g. "B90"
    opening_name TEXT, -- e.g. "Sicilian Defense: Najdorf Variation"
    opening_url TEXT, -- chess.com opening explorer link
    analysis_status TEXT DEFAULT 'pending', -- pending, processing, completed, failed
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Position analysis results
CREATE TABLE IF NOT EXISTS positions (
    id INTEGER PRIMARY KEY,
    game_id INTEGER NOT NULL REFERENCES games(id),
    move_number INTEGER NOT NULL,
    fen TEXT NOT NULL,
    move_played TEXT NOT NULL,
    best_move TEXT NOT NULL,
    eval_before REAL, -- centipawns (+ white advantage)
    eval_after REAL,
    eval_diff REAL, -- difference (negative = blunder)
    classification TEXT, -- blunder, mistake, inaccuracy, missed_win, good, excellent
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Flashcards for training
CREATE TABLE IF NOT EXISTS flashcards (
    id INTEGER PRIMARY KEY,
    position_id INTEGER NOT NULL REFERENCES positions(id),
    due_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    interval_days INTEGER DEFAULT 0,
    ease_factor REAL DEFAULT 2.5,
    times_reviewed INTEGER DEFAULT 0,
    times_correct INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Opening performance statistics (view)
CREATE VIEW IF NOT EXISTS opening_stats AS
SELECT 
    profile_id,
    opening_name,
    eco_code,
    COUNT(*) as total_games,
    SUM(CASE WHEN result = 'win' THEN 1 ELSE 0 END) as wins,
    SUM(CASE WHEN result = 'draw' THEN 1 ELSE 0 END) as draws,
    SUM(CASE WHEN result = 'loss' THEN 1 ELSE 0 END) as losses,
    ROUND(100.0 * SUM(CASE WHEN result = 'win' THEN 1 ELSE 0 END) / COUNT(*), 1) as win_rate,
    AVG((SELECT COUNT(*) FROM positions p WHERE p.game_id = games.id AND p.classification = 'blunder')) as avg_blunders
FROM games
WHERE opening_name IS NOT NULL
GROUP BY profile_id, opening_name, eco_code;

