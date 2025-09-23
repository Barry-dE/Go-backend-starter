package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"strconv"

	"github.com/Barry-dE/go-backend-boilerplate/internal/config"
	"github.com/jackc/pgx/v5"
	tern "github.com/jackc/tern/v2/migrate"
	"github.com/rs/zerolog"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

func Migrate(ctx context.Context, logger *zerolog.Logger, cfg *config.Config) error {
	hostPort := net.JoinHostPort(cfg.Database.Host, strconv.Itoa(cfg.Database.Port))

	// URL-encode the database password
	password := url.QueryEscape(cfg.Database.Password)
	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s", cfg.Database.User, password, hostPort, cfg.Database.Name, cfg.Database.SSLMode)

	// Use a single database connection for migrations.
	dbConn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return err
	}

	// Close DB connection when migration is finish.
	defer dbConn.Close(ctx)

	// Create a new migrator instance with the database connection and the schema version table name.
	migrator, err := tern.NewMigrator(ctx, dbConn, "schema_version")
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	// Access the "migrations" subdirectory from the embedded filesystem
	fsImplementation, err := fs.Sub(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to get sub filesystem: %w", err)
	}

	// Load all SQL migration files into the migrator.
	if err := migrator.LoadMigrations(fsImplementation); err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Get the current migration version before applying new migrations.
	version, err := migrator.GetCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current migration version: %w", err)
	}

	// Apply all pending migrations to update the database schema.
	if err := migrator.Migrate(ctx); err != nil {
		return err
	}

	// Log the migration result.
	// If the version hasn't changed, the database was already up to date.
	// Otherwise, log the old and new version numbers.
	if version == int32(len(migrator.Migrations)) {
		logger.Info().Msgf("Database is up to date at version %d", len(migrator.Migrations))
	} else {
		logger.Info().Msgf("Database migrated from version %d to %d", version, len(migrator.Migrations))
	}
	return nil
}
