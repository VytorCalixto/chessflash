package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/vytor/chessflash/internal/logger"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type DB struct {
	*sql.DB
	log *logger.Logger
}

func Open(path string) (*DB, error) {
	log := logger.Default().WithPrefix("db")

	dsn := fmt.Sprintf("%s?_busy_timeout=5000&_foreign_keys=on&_journal_mode=WAL&_synchronous=NORMAL", path)
	log.Info("opening database: %s", path)

	sqlDB, err := sql.Open("sqlite3", dsn)
	if err != nil {
		log.Error("failed to open database: %v", err)
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1) // SQLite best practice for single writer

	db := &DB{DB: sqlDB, log: log}

	log.Debug("applying migrations")
	if err := db.applyMigrations(context.Background()); err != nil {
		log.Error("failed to apply migrations: %v", err)
		return nil, err
	}

	log.Info("database ready")
	return db, nil
}

func (db *DB) applyMigrations(ctx context.Context) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at DATETIME DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		return err
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		version := entry.Name()
		applied, err := db.isMigrationApplied(ctx, version)
		if err != nil {
			return err
		}
		if applied {
			db.log.Debug("migration %s already applied, skipping", version)
			continue
		}
		sqlBytes, err := migrationsFS.ReadFile("migrations/" + version)
		if err != nil {
			return err
		}
		db.log.Info("applying migration: %s", version)
		if _, err := db.ExecContext(ctx, string(sqlBytes)); err != nil {
			db.log.Error("migration %s failed: %v", version, err)
			return fmt.Errorf("apply migration %s: %w", version, err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES (?)`, version); err != nil {
			return err
		}
		db.log.Info("migration %s applied successfully", version)
	}
	return nil
}

func (db *DB) isMigrationApplied(ctx context.Context, version string) (bool, error) {
	var v string
	err := db.QueryRowContext(ctx, `SELECT version FROM schema_migrations WHERE version = ?`, version).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func tx(ctx context.Context, db *DB, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		db.log.Error("failed to begin transaction: %v", err)
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		db.log.Debug("transaction rolled back due to error: %v", err)
		return err
	}
	if err := tx.Commit(); err != nil {
		db.log.Error("failed to commit transaction: %v", err)
		return err
	}
	db.log.Debug("transaction committed")
	return nil
}

// Helper to build filter clauses.
func whereParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	return "WHERE " + strings.Join(parts, " AND ")
}
