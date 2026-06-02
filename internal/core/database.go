package core

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// OpenDatabase opens (or creates) the SQLite database and runs migrations.
func OpenDatabase(dbPath string) (*sql.DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory %s: %w", dir, err)
	}

	// Open SQLite connection
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_foreign_keys=on", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Run migrations
	if err := runMigrations(db, dbPath); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return db, nil
}

// runMigrations applies embedded SQL migrations to the database.
func runMigrations(db *sql.DB, dbPath string) error {
	driver, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://internal/core/migrations",
		"sqlite",
		driver,
	)
	if err != nil {
		// If the migration source directory doesn't exist, run inline migration
		return runInlineMigrations(db)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration up failed: %w", err)
	}

	return nil
}

// runInlineMigrations creates tables directly when external migration files
// are not available.
func runInlineMigrations(db *sql.DB) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS isos (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			name        TEXT    NOT NULL,
			source_path TEXT    NOT NULL UNIQUE,
			size        INTEGER NOT NULL DEFAULT 0,
			sha256      TEXT    NOT NULL DEFAULT '',
			architecture TEXT   NOT NULL DEFAULT '',
			boot_modes  TEXT    NOT NULL DEFAULT '[]',
			distro      TEXT    NOT NULL DEFAULT '',
			version     TEXT    NOT NULL DEFAULT '',
			boot_profile TEXT   NOT NULL DEFAULT '',
			cached_path TEXT    NOT NULL DEFAULT '',
			status      TEXT    NOT NULL DEFAULT 'new',
			last_scanned DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			dirty   INTEGER NOT NULL DEFAULT 0
		)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("failed to execute migration: %w\nSQL: %s", err, m)
		}
	}

	return nil
}
