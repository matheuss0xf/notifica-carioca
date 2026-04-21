package postgres

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Migrator runs SQL migrations from a configured source into PostgreSQL.
type Migrator struct {
	sourceURL string
}

// NewMigrator creates a migration adapter.
func NewMigrator(sourceURL string) *Migrator {
	return &Migrator{sourceURL: sourceURL}
}

// Up applies all pending migrations.
func (m *Migrator) Up(databaseURL string) error {
	instance, err := migrate.New(m.sourceURL, databaseURL)
	if err != nil {
		return fmt.Errorf("creating migration instance: %w", err)
	}
	defer instance.Close()

	if err := instance.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("running migrations: %w", err)
	}

	return nil
}
