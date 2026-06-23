package db

import "database/sql"

// DB wraps the underlying SQL database connection.
type DB struct {
	conn *sql.DB
}

// Init initializes the database connection and runs migrations.
func Init(dataDir string) (*DB, error) {
	// TODO: Phase 1 — open SQLite, run migrations
	return &DB{}, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	// TODO: Phase 1 — close connection
	return nil
}
