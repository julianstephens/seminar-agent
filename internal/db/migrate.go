package db

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	// pgx/v5 driver registers the "pgx5" scheme with golang-migrate.
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/julianstephens/formation/internal/config"
	"github.com/julianstephens/formation/migrations"
)

// Migrate runs all pending up migrations against the database.
// It is idempotent — already-applied migrations are silently skipped.
func Migrate(cfg *config.Config) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("open migration source: %w", err)
	}

	// golang-migrate's pgx/v5 driver expects the "pgx5://" URL scheme.
	dbURL := toPgx5URL(cfg.DatabaseURL)

	m, err := migrate.NewWithSourceInstance("iofs", src, dbURL)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer (func() {
		if _, err := m.Close(); err != nil {
			fmt.Printf("close migrator: %v\n", err)
		}
	})()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

// toPgx5URL replaces the postgres/postgresql URL scheme with "pgx5" so
// golang-migrate routes to the correct pgx v5 driver.
func toPgx5URL(dsn string) string {
	for _, scheme := range []string{"postgresql://", "postgres://"} {
		if strings.HasPrefix(dsn, scheme) {
			return "pgx5://" + dsn[len(scheme):]
		}
	}
	return dsn
}
