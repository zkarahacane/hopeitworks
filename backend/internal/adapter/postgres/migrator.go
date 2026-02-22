package postgres

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // pgx/v5 database driver
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// RunMigrations applies all pending database migrations using the provided
// embedded filesystem and DSN. It logs the outcome: how many migrations were
// applied, or that the schema is already up to date.
//
// Returns an error if any migration fails, wrapping context about the failure.
func RunMigrations(migrationsFS fs.FS, dsn string, logger *slog.Logger) error {
	source, err := iofs.New(migrationsFS, ".")
	if err != nil {
		return fmt.Errorf("creating migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, "pgx5://"+dsn[len("postgres://"):])
	if err != nil {
		return fmt.Errorf("creating migrate instance: %w", err)
	}
	defer func() {
		_, _ = m.Close()
	}()

	// Read current version before migrating to calculate applied count
	versionBefore, _, _ := m.Version()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Info("database schema up to date")
			return nil
		}
		return fmt.Errorf("applying migrations: %w", err)
	}

	versionAfter, _, _ := m.Version()
	logger.Info("migrations applied", "from_version", versionBefore, "to_version", versionAfter)

	return nil
}
