package testutil

import (
	"database/sql"
	"embed"
	"testing"

	"github.com/stretchr/testify/require"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var testMigrationsFS embed.FS

// NewTestDB creates an in-memory SQLite database with all migrations applied.
// The database is configured with foreign keys enabled and WAL mode.
func NewTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on&_journal_mode=WAL")
	require.NoError(t, err)

	// Apply migrations
	migrations := []string{
		"migrations/0001_init.sql",
		"migrations/0002_cascade_delete.sql",
		"migrations/0003_add_mate_fields.sql",
		"migrations/0004_stats_cache.sql",
		"migrations/0005_review_history.sql",
		"migrations/0006_add_performance_indexes.sql",
	}

	for _, migration := range migrations {
		sqlBytes, err := testMigrationsFS.ReadFile(migration)
		require.NoError(t, err, "failed to read migration %s", migration)

		_, err = db.Exec(string(sqlBytes))
		require.NoError(t, err, "failed to apply migration %s", migration)
	}

	return db
}

// MustClose closes a resource and fails the test on error.
func MustClose(t *testing.T, closer interface{ Close() error }) {
	require.NoError(t, closer.Close())
}
