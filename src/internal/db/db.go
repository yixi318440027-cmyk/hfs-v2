package db

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB wraps the underlying SQL database connection.
type DB struct {
	conn *sql.DB
}

// Init initializes the database connection, runs PRAGMAs, and applies migrations.
func Init(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dataDir, "hfs.db")
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Enable WAL mode for better concurrent read performance.
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, err
	}

	// Enable foreign key enforcement.
	if _, err := conn.Exec("PRAGMA foreign_keys=ON"); err != nil {
		conn.Close()
		return nil, err
	}

	d := &DB{conn: conn}

	if err := d.migrate(); err != nil {
		conn.Close()
		return nil, err
	}

	return d, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	if d.conn == nil {
		return nil
	}
	return d.conn.Close()
}

// Conn returns the underlying *sql.DB for direct queries.
func (d *DB) Conn() *sql.DB {
	return d.conn
}

// migrate creates tables and inserts default data.
func (d *DB) migrate() error {
	_, err := d.conn.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user',
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
	`)
	if err != nil {
		return err
	}

	// Insert default admin user. The hash is a bcrypt hash of "admin" (cost=12).
	// Generated separately; this placeholder will be replaced at startup via the auth layer.
	// We use INSERT OR IGNORE to skip if the admin user already exists.
	// The actual hash is set by the application after the auth service is initialized.
	_, err = d.conn.Exec(`
		INSERT OR IGNORE INTO users (username, password_hash, role)
		VALUES ('admin', '', 'admin');
	`)
	if err != nil {
		return err
	}

	return nil
}
