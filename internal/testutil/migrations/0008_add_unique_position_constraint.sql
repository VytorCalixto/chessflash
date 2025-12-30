-- Update flashcards that reference duplicate positions to point to the kept position (lowest id)
UPDATE flashcards
SET position_id = (
    SELECT MIN(p2.id)
    FROM positions p2
    WHERE p2.game_id = (SELECT game_id FROM positions WHERE id = flashcards.position_id)
    AND p2.move_number = (SELECT move_number FROM positions WHERE id = flashcards.position_id)
)
WHERE position_id IN (
    SELECT id
    FROM positions
    WHERE id NOT IN (
        SELECT MIN(id)
        FROM positions
        GROUP BY game_id, move_number
    )
);

-- Remove duplicate positions, keeping the one with the lowest id for each (game_id, move_number) pair
DELETE FROM positions
WHERE id NOT IN (
    SELECT MIN(id)
    FROM positions
    GROUP BY game_id, move_number
);

-- Add unique constraint on (game_id, move_number)
CREATE UNIQUE INDEX IF NOT EXISTS idx_positions_game_id_move_number_unique 
ON positions(game_id, move_number);
