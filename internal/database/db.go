package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB wraps standard sql.DB with concurrency safety and helper methods.
type DB struct {
	conn *sql.DB
	mu   sync.RWMutex
}

// Open creates or connects to a SQLite database and applies migrations.
func Open(dataSourceName string) (*DB, error) {
	// Enable WAL mode, busy timeout, and foreign keys for high concurrency and consistency
	dsn := dataSourceName
	if !strings.Contains(dsn, "?") {
		dsn += "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	} else {
		dsn += "&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	}

	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Bounded connection pool for SQLite to prevent lock contention
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(time.Hour)

	db := &DB{conn: conn}
	if err := db.migrate(context.Background()); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.conn.Close()
}

// Raw returns the underlying *sql.DB.
func (d *DB) Raw() *sql.DB {
	return d.conn
}

// migrate executes pending migration files in alphanumeric order.
func (d *DB) migrate(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Ensure schema_migrations table exists
	_, err := d.conn.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TEXT NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("could not create migrations table: %w", err)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("could not read migrations directory: %w", err)
	}

	type migFile struct {
		version int
		name    string
	}
	var migFiles []migFile

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		parts := strings.SplitN(entry.Name(), "_", 2)
		if len(parts) < 2 {
			continue
		}
		v, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		migFiles = append(migFiles, migFile{version: v, name: entry.Name()})
	}

	sort.Slice(migFiles, func(i, j int) bool {
		return migFiles[i].version < migFiles[j].version
	})

	for _, m := range migFiles {
		var exists int
		err := d.conn.QueryRowContext(ctx, "SELECT COUNT(1) FROM schema_migrations WHERE version = ?", m.version).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed querying migration %d: %w", m.version, err)
		}
		if exists > 0 {
			continue
		}

		content, err := migrationsFS.ReadFile("migrations/" + m.name)
		if err != nil {
			return fmt.Errorf("could not read migration %s: %w", m.name, err)
		}

		tx, err := d.conn.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed starting tx for migration %d: %w", m.version, err)
		}

		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed executing migration %s: %w", m.name, err)
		}

		nowStr := time.Now().UTC().Format(time.RFC3339)
		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)", m.version, m.name, nowStr); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed recording migration %d: %w", m.version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed committing migration %d: %w", m.version, err)
		}
	}

	return nil
}
