-- Add rating columns to games for tracking player/opponent strength.
ALTER TABLE games ADD COLUMN player_rating INTEGER;
ALTER TABLE games ADD COLUMN opponent_rating INTEGER;

-- Cache tables for precomputed statistics.
CREATE TABLE IF NOT EXISTS opening_stats_cache (
    profile_id INTEGER NOT NULL,
    opening_name TEXT NOT NULL,
    eco_code TEXT,
    total_games INTEGER DEFAULT 0,
    wins INTEGER DEFAULT 0,
    draws INTEGER DEFAULT 0,
    losses INTEGER DEFAULT 0,
    win_rate REAL DEFAULT 0,
    avg_blunders REAL DEFAULT 0,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (profile_id, opening_name)
);

CREATE TABLE IF NOT EXISTS opponent_stats_cache (
    profile_id INTEGER NOT NULL,
    opponent TEXT NOT NULL,
    total_games INTEGER DEFAULT 0,
    wins INTEGER DEFAULT 0,
    draws INTEGER DEFAULT 0,
    losses INTEGER DEFAULT 0,
    win_rate REAL DEFAULT 0,
    avg_opponent_rating REAL,
    last_played_at DATETIME,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (profile_id, opponent)
);

CREATE TABLE IF NOT EXISTS time_class_stats_cache (
    profile_id INTEGER NOT NULL,
    time_class TEXT NOT NULL,
    total_games INTEGER DEFAULT 0,
    wins INTEGER DEFAULT 0,
    draws INTEGER DEFAULT 0,
    losses INTEGER DEFAULT 0,
    win_rate REAL DEFAULT 0,
    avg_blunders REAL DEFAULT 0,
    avg_game_length REAL DEFAULT 0,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (profile_id, time_class)
);

CREATE TABLE IF NOT EXISTS color_stats_cache (
    profile_id INTEGER NOT NULL,
    played_as TEXT NOT NULL,
    total_games INTEGER DEFAULT 0,
    wins INTEGER DEFAULT 0,
    draws INTEGER DEFAULT 0,
    losses INTEGER DEFAULT 0,
    win_rate REAL DEFAULT 0,
    avg_blunders REAL DEFAULT 0,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (profile_id, played_as)
);

CREATE TABLE IF NOT EXISTS monthly_stats_cache (
    profile_id INTEGER NOT NULL,
    year_month TEXT NOT NULL,
    total_games INTEGER DEFAULT 0,
    wins INTEGER DEFAULT 0,
    draws INTEGER DEFAULT 0,
    losses INTEGER DEFAULT 0,
    win_rate REAL DEFAULT 0,
    total_blunders INTEGER DEFAULT 0,
    blunder_rate REAL DEFAULT 0,
    avg_rating REAL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (profile_id, year_month)
);

CREATE TABLE IF NOT EXISTS mistake_phase_cache (
    profile_id INTEGER NOT NULL,
    phase TEXT NOT NULL,
    classification TEXT NOT NULL,
    count INTEGER DEFAULT 0,
    avg_eval_loss REAL DEFAULT 0,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (profile_id, phase, classification)
);

CREATE TABLE IF NOT EXISTS rating_stats_cache (
    profile_id INTEGER NOT NULL,
    time_class TEXT NOT NULL,
    min_rating INTEGER,
    max_rating INTEGER,
    avg_rating REAL,
    current_rating INTEGER,
    rating_change INTEGER,
    games_tracked INTEGER DEFAULT 0,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (profile_id, time_class)
);

-- Seed cache tables from existing data (best-effort, may be empty if no data yet).

INSERT INTO opening_stats_cache (profile_id, opening_name, eco_code, total_games, wins, draws, losses, win_rate, avg_blunders)
SELECT g.profile_id,
       g.opening_name,
       g.eco_code,
       COUNT(*) AS total_games,
       SUM(CASE WHEN g.result = 'win' THEN 1 ELSE 0 END) AS wins,
       SUM(CASE WHEN g.result = 'draw' THEN 1 ELSE 0 END) AS draws,
       SUM(CASE WHEN g.result = 'loss' THEN 1 ELSE 0 END) AS losses,
       ROUND(100.0 * SUM(CASE WHEN g.result = 'win' THEN 1 ELSE 0 END) / COUNT(*), 1) AS win_rate,
       COALESCE(AVG(COALESCE(b.blunder_count, 0)), 0) AS avg_blunders
FROM games g
LEFT JOIN (
    SELECT game_id, COUNT(*) AS blunder_count
    FROM positions
    WHERE classification = 'blunder'
    GROUP BY game_id
) b ON b.game_id = g.id
WHERE g.opening_name IS NOT NULL
GROUP BY g.profile_id, g.opening_name, g.eco_code;

INSERT INTO opponent_stats_cache (profile_id, opponent, total_games, wins, draws, losses, win_rate, avg_opponent_rating, last_played_at)
SELECT g.profile_id,
       g.opponent,
       COUNT(*) AS total_games,
       SUM(CASE WHEN g.result = 'win' THEN 1 ELSE 0 END) AS wins,
       SUM(CASE WHEN g.result = 'draw' THEN 1 ELSE 0 END) AS draws,
       SUM(CASE WHEN g.result = 'loss' THEN 1 ELSE 0 END) AS losses,
       ROUND(100.0 * SUM(CASE WHEN g.result = 'win' THEN 1 ELSE 0 END) / COUNT(*), 1) AS win_rate,
       AVG(g.opponent_rating) AS avg_opponent_rating,
       MAX(g.played_at) AS last_played_at
FROM games g
GROUP BY g.profile_id, g.opponent;

INSERT INTO time_class_stats_cache (profile_id, time_class, total_games, wins, draws, losses, win_rate, avg_blunders, avg_game_length)
SELECT g.profile_id,
       g.time_class,
       COUNT(*) AS total_games,
       SUM(CASE WHEN g.result = 'win' THEN 1 ELSE 0 END) AS wins,
       SUM(CASE WHEN g.result = 'draw' THEN 1 ELSE 0 END) AS draws,
       SUM(CASE WHEN g.result = 'loss' THEN 1 ELSE 0 END) AS losses,
       ROUND(100.0 * SUM(CASE WHEN g.result = 'win' THEN 1 ELSE 0 END) / COUNT(*), 1) AS win_rate,
       COALESCE(AVG(COALESCE(b.blunder_count, 0)), 0) AS avg_blunders,
       COALESCE(AVG(COALESCE(m.moves_played, 0)), 0) AS avg_game_length
FROM games g
LEFT JOIN (
    SELECT game_id, COUNT(*) AS blunder_count
    FROM positions
    WHERE classification = 'blunder'
    GROUP BY game_id
) b ON b.game_id = g.id
LEFT JOIN (
    SELECT game_id, MAX(move_number) AS moves_played
    FROM positions
    GROUP BY game_id
) m ON m.game_id = g.id
GROUP BY g.profile_id, g.time_class;

INSERT INTO color_stats_cache (profile_id, played_as, total_games, wins, draws, losses, win_rate, avg_blunders)
SELECT g.profile_id,
       g.played_as,
       COUNT(*) AS total_games,
       SUM(CASE WHEN g.result = 'win' THEN 1 ELSE 0 END) AS wins,
       SUM(CASE WHEN g.result = 'draw' THEN 1 ELSE 0 END) AS draws,
       SUM(CASE WHEN g.result = 'loss' THEN 1 ELSE 0 END) AS losses,
       ROUND(100.0 * SUM(CASE WHEN g.result = 'win' THEN 1 ELSE 0 END) / COUNT(*), 1) AS win_rate,
       COALESCE(AVG(COALESCE(b.blunder_count, 0)), 0) AS avg_blunders
FROM games g
LEFT JOIN (
    SELECT game_id, COUNT(*) AS blunder_count
    FROM positions
    WHERE classification = 'blunder'
    GROUP BY game_id
) b ON b.game_id = g.id
GROUP BY g.profile_id, g.played_as;

INSERT INTO monthly_stats_cache (profile_id, year_month, total_games, wins, draws, losses, win_rate, total_blunders, blunder_rate, avg_rating)
SELECT g.profile_id,
       strftime('%Y-%m', g.played_at) AS year_month,
       COUNT(*) AS total_games,
       SUM(CASE WHEN g.result = 'win' THEN 1 ELSE 0 END) AS wins,
       SUM(CASE WHEN g.result = 'draw' THEN 1 ELSE 0 END) AS draws,
       SUM(CASE WHEN g.result = 'loss' THEN 1 ELSE 0 END) AS losses,
       ROUND(100.0 * SUM(CASE WHEN g.result = 'win' THEN 1 ELSE 0 END) / COUNT(*), 1) AS win_rate,
       COALESCE(SUM(COALESCE(b.blunder_count, 0)), 0) AS total_blunders,
       CASE WHEN COUNT(*) > 0 THEN ROUND(1.0 * COALESCE(SUM(COALESCE(b.blunder_count, 0)), 0) / COUNT(*), 3) ELSE 0 END AS blunder_rate,
       AVG(g.player_rating) AS avg_rating
FROM games g
LEFT JOIN (
    SELECT game_id, COUNT(*) AS blunder_count
    FROM positions
    WHERE classification = 'blunder'
    GROUP BY game_id
) b ON b.game_id = g.id
GROUP BY g.profile_id, year_month;

INSERT INTO mistake_phase_cache (profile_id, phase, classification, count, avg_eval_loss)
SELECT g.profile_id,
       CASE
           WHEN p.move_number <= 15 THEN 'opening'
           WHEN p.move_number <= 35 THEN 'middlegame'
           ELSE 'endgame'
       END AS phase,
       p.classification,
       COUNT(*) AS count,
       AVG(CASE WHEN p.eval_diff < 0 THEN -p.eval_diff ELSE 0 END) AS avg_eval_loss
FROM positions p
JOIN games g ON g.id = p.game_id
GROUP BY g.profile_id, phase, p.classification;

INSERT INTO rating_stats_cache (profile_id, time_class, min_rating, max_rating, avg_rating, current_rating, rating_change, games_tracked)
SELECT g.profile_id,
       g.time_class,
       MIN(g.player_rating) AS min_rating,
       MAX(g.player_rating) AS max_rating,
       AVG(g.player_rating) AS avg_rating,
       (
           SELECT g2.player_rating
           FROM games g2
           WHERE g2.profile_id = g.profile_id AND g2.time_class = g.time_class AND g2.player_rating IS NOT NULL
           ORDER BY g2.played_at DESC
           LIMIT 1
       ) AS current_rating,
       (
           COALESCE((
               SELECT g2.player_rating
               FROM games g2
               WHERE g2.profile_id = g.profile_id AND g2.time_class = g.time_class AND g2.player_rating IS NOT NULL
               ORDER BY g2.played_at DESC
               LIMIT 1
           ), 0) -
           COALESCE((
               SELECT g3.player_rating
               FROM games g3
               WHERE g3.profile_id = g.profile_id AND g3.time_class = g.time_class AND g3.player_rating IS NOT NULL
               ORDER BY g3.played_at ASC
               LIMIT 1
           ), 0)
       ) AS rating_change,
       COUNT(g.player_rating) AS games_tracked
FROM games g
WHERE g.player_rating IS NOT NULL
GROUP BY g.profile_id, g.time_class;

-- Replace old view with cached approach.
DROP VIEW IF EXISTS opening_stats;
