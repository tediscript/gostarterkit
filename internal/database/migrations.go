package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Migration represents a database migration
type Migration struct {
	Version int
	Up      string
	Down    string
}

// MigrationRunner handles running database migrations
type MigrationRunner struct {
	db            *Database
	migrationsDir string
}

// NewMigrationRunner creates a new migration runner
func NewMigrationRunner(db *Database, migrationsDir string) *MigrationRunner {
	return &MigrationRunner{db: db, migrationsDir: migrationsDir}
}

// RunMigrations is a convenience function to run all pending migrations
func RunMigrations(db *Database, migrationsDir string) error {
	runner := NewMigrationRunner(db, migrationsDir)
	return runner.Migrate(context.Background())
}

// Migrate runs all pending migrations
func (m *MigrationRunner) Migrate(ctx context.Context) error {
	// Create migrations table if it doesn't exist
	if err := m.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Load migrations from files
	migrations, err := m.loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Get current migration version
	currentVersion, err := m.getCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current migration version: %w", err)
	}

	// Run pending migrations
	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue
		}

		if err := m.runMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to run migration %d: %w", migration.Version, err)
		}
	}

	return nil
}

// Rollback rolls back the last migration
func (m *MigrationRunner) Rollback(ctx context.Context) error {
	// Create migrations table if it doesn't exist
	if err := m.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current migration version
	currentVersion, err := m.getCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current migration version: %w", err)
	}

	if currentVersion == 0 {
		return fmt.Errorf("no migrations to rollback")
	}

	// Load migrations from files
	migrations, err := m.loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Find migration to rollback
	var migration *Migration
	for i := len(migrations) - 1; i >= 0; i-- {
		if migrations[i].Version == currentVersion {
			migration = &migrations[i]
			break
		}
	}

	if migration == nil {
		return fmt.Errorf("migration version %d not found", currentVersion)
	}

	// Run down migration
	if _, err := m.db.Exec(ctx, migration.Down); err != nil {
		return fmt.Errorf("failed to rollback migration %d: %w", migration.Version, err)
	}

	// Delete migration record
	if _, err := m.db.Exec(ctx, "DELETE FROM schema_migrations WHERE version = ?", migration.Version); err != nil {
		return fmt.Errorf("failed to delete migration record: %w", err)
	}

	return nil
}

// createMigrationsTable creates the schema_migrations table if it doesn't exist
func (m *MigrationRunner) createMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`
	if _, err := m.db.Exec(ctx, query); err != nil {
		return err
	}
	return nil
}

// getCurrentVersion returns the current migration version
func (m *MigrationRunner) getCurrentVersion(ctx context.Context) (int, error) {
	var version int
	err := m.db.QueryRow(ctx, "SELECT MAX(version) FROM schema_migrations").Scan(&version)
	if err != nil {
		// No migrations yet
		if strings.Contains(err.Error(), "no rows in result set") {
			return 0, nil
		}
		return 0, err
	}
	return version, nil
}

// loadMigrations loads migration files from the migrations directory
func (m *MigrationRunner) loadMigrations() ([]Migration, error) {
	migrationsDir := m.migrationsDir

	// Check if migrations directory exists
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		return []Migration{}, nil
	}

	// Read migration files
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, err
	}

	// Group migrations by version
	migrationsMap := make(map[int]Migration)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()
		if !strings.HasSuffix(filename, ".sql") {
			continue
		}

		// Parse filename: version_name.up.sql or version_name.down.sql
		parts := strings.Split(filename, "_")
		if len(parts) < 3 {
			continue
		}

		var version int
		if _, err := fmt.Sscanf(parts[0], "%d", &version); err != nil {
			continue
		}

		// Read file content
		content, err := os.ReadFile(filepath.Join(migrationsDir, filename))
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		// Store up or down migration
		migration := migrationsMap[version]
		migration.Version = version

		if strings.Contains(filename, ".up.sql") {
			migration.Up = string(content)
		} else if strings.Contains(filename, ".down.sql") {
			migration.Down = string(content)
		}

		migrationsMap[version] = migration
	}

	// Convert map to slice and sort by version
	migrations := make([]Migration, 0, len(migrationsMap))
	for _, m := range migrationsMap {
		migrations = append(migrations, m)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// runMigration runs a single migration
func (m *MigrationRunner) runMigration(ctx context.Context, migration Migration) error {
	// Begin transaction
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// Ensure transaction is rolled back if error occurs
	defer tx.Rollback()

	// Execute up migration
	if _, err := tx.ExecContext(ctx, migration.Up); err != nil {
		return err
	}

	// Record migration
	if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES (?)", migration.Version); err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
