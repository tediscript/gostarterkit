package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/tediscript/gostarterkit/internal/config"
	_ "modernc.org/sqlite" // SQLite driver
)

// Database wraps the sql.DB with additional functionality
type Database struct {
	db   *sql.DB
	lock sync.RWMutex
}

// New creates a new database connection pool
func New(cfg *config.Config) (*Database, error) {
	// Ensure database directory exists
	dbPath := cfg.SQLite.DBFile
	dbDir := filepath.Dir(dbPath)
	if dbDir != "." && dbDir != "" {
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	// Open database connection
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.SQLite.MaxOpenConnections)
	db.SetMaxIdleConns(cfg.SQLite.MaxIdleConnections)
	db.SetConnMaxLifetime(time.Duration(cfg.SQLite.ConnectionMaxLifetimeSeconds) * time.Second)

	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys=ON;"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	return &Database{db: db}, nil
}

// DB returns the underlying sql.DB instance
func (d *Database) DB() *sql.DB {
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.db
}

// Ping verifies the database connection is alive
func (d *Database) Ping(ctx context.Context) error {
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.db.PingContext(ctx)
}

// Close closes the database connection pool
func (d *Database) Close() error {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// BeginTx starts a transaction
func (d *Database) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.db.BeginTx(ctx, opts)
}

// Exec executes a query without returning any rows
func (d *Database) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.db.ExecContext(ctx, query, args...)
}

// Query executes a query that returns rows
func (d *Database) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.db.QueryContext(ctx, query, args...)
}

// QueryRow executes a query that returns at most one row
func (d *Database) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.db.QueryRowContext(ctx, query, args...)
}

// Stats returns connection pool statistics
func (d *Database) Stats() sql.DBStats {
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.db.Stats()
}
