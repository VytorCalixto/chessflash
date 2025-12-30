-- Remove duplicate flashcards, keeping the one with the lowest id for each position_id
-- Related records in review_history and puzzle_rush_attempts will be automatically deleted
-- due to ON DELETE CASCADE constraints
DELETE FROM flashcards
WHERE id NOT IN (
    SELECT MIN(id)
    FROM flashcards
    GROUP BY position_id
);

-- Add unique constraint on position_id
CREATE UNIQUE INDEX IF NOT EXISTS idx_flashcards_position_id_unique 
ON flashcards(position_id);
