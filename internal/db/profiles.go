package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/vytor/chessflash/internal/logger"
	"github.com/vytor/chessflash/internal/models"
)

func (db *DB) UpsertProfile(ctx context.Context, username string) (*models.Profile, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("upserting profile for username: %s", username)

	var p models.Profile
	err := db.QueryRowContext(ctx, `
INSERT INTO profiles (username)
VALUES (?)
ON CONFLICT(username) DO UPDATE SET username = excluded.username
RETURNING id, username, created_at, last_sync_at
`, username).Scan(&p.ID, &p.Username, &p.CreatedAt, &p.LastSyncAt)
	if err != nil {
		log.Error("failed to upsert profile: %v", err)
		return nil, err
	}
	log.Debug("profile upserted: id=%d", p.ID)
	return &p, nil
}

func (db *DB) UpdateProfileSync(ctx context.Context, id int64, t time.Time) error {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("updating profile sync time: profile_id=%d", id)

	_, err := db.ExecContext(ctx, `UPDATE profiles SET last_sync_at = ? WHERE id = ?`, t, id)
	if err != nil {
		log.Error("failed to update profile sync: %v", err)
	}
	return err
}

func (db *DB) ListProfiles(ctx context.Context) ([]models.Profile, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("listing profiles")

	rows, err := db.QueryContext(ctx, `
SELECT id, username, created_at, last_sync_at
FROM profiles
ORDER BY created_at ASC
`)
	if err != nil {
		log.Error("failed to list profiles: %v", err)
		return nil, err
	}
	defer rows.Close()

	var profiles []models.Profile
	for rows.Next() {
		var p models.Profile
		if err := rows.Scan(&p.ID, &p.Username, &p.CreatedAt, &p.LastSyncAt); err != nil {
			log.Error("failed to scan profile row: %v", err)
			return nil, err
		}
		profiles = append(profiles, p)
	}

	log.Debug("found %d profiles", len(profiles))
	return profiles, rows.Err()
}

func (db *DB) GetProfile(ctx context.Context, id int64) (*models.Profile, error) {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("getting profile: id=%d", id)

	var p models.Profile
	err := db.QueryRowContext(ctx, `
SELECT id, username, created_at, last_sync_at
FROM profiles
WHERE id = ?
`, id).Scan(&p.ID, &p.Username, &p.CreatedAt, &p.LastSyncAt)
	if errors.Is(err, sql.ErrNoRows) {
		log.Debug("profile not found: id=%d", id)
		return nil, nil
	}
	if err != nil {
		log.Error("failed to get profile: %v", err)
		return nil, err
	}
	return &p, nil
}

func (db *DB) DeleteProfile(ctx context.Context, id int64) error {
	log := logger.FromContext(ctx).WithPrefix("db")
	log.Debug("deleting profile and related data: id=%d", id)

	return tx(ctx, db, func(tx *sql.Tx) error {
		// Delete flashcards -> positions -> games -> profile to respect FK constraints.
		if _, err := tx.ExecContext(ctx, `
DELETE FROM flashcards
WHERE position_id IN (
    SELECT p.id FROM positions p
    JOIN games g ON g.id = p.game_id
    WHERE g.profile_id = ?
)
`, id); err != nil {
			log.Error("failed to delete flashcards for profile %d: %v", id, err)
			return err
		}

		if _, err := tx.ExecContext(ctx, `
DELETE FROM positions
WHERE game_id IN (SELECT id FROM games WHERE profile_id = ?)
`, id); err != nil {
			log.Error("failed to delete positions for profile %d: %v", id, err)
			return err
		}

		if _, err := tx.ExecContext(ctx, `DELETE FROM games WHERE profile_id = ?`, id); err != nil {
			log.Error("failed to delete games for profile %d: %v", id, err)
			return err
		}

		if _, err := tx.ExecContext(ctx, `DELETE FROM profiles WHERE id = ?`, id); err != nil {
			log.Error("failed to delete profile %d: %v", id, err)
			return err
		}

		log.Debug("profile %d deleted with cascading data", id)
		return nil
	})
}
