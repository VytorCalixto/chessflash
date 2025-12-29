package sqlite

import (
	"context"
	"database/sql"
	"strings"

	"github.com/vytor/chessflash/internal/logger"
)

// Helper functions shared across repository implementations

func whereParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	return "WHERE " + strings.Join(parts, " AND ")
}

func tx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	log := logger.FromContext(ctx).WithPrefix("repo")
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Error("failed to begin transaction: %v", err)
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		log.Debug("transaction rolled back due to error: %v", err)
		return err
	}
	if err := tx.Commit(); err != nil {
		log.Error("failed to commit transaction: %v", err)
		return err
	}
	log.Debug("transaction committed")
	return nil
}
