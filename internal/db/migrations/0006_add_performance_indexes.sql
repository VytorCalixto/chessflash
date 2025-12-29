-- Performance indexes for frequently queried columns
CREATE INDEX IF NOT EXISTS idx_games_profile_id_played_at ON games(profile_id, played_at);
CREATE INDEX IF NOT EXISTS idx_positions_game_id ON positions(game_id);
CREATE INDEX IF NOT EXISTS idx_positions_classification ON positions(classification);
CREATE INDEX IF NOT EXISTS idx_flashcards_position_id ON flashcards(position_id);
CREATE INDEX IF NOT EXISTS idx_flashcards_due_at ON flashcards(due_at);
