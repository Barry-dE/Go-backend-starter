package testing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Barry-dE/go-backend-boilerplate/internal/config"
	"github.com/Barry-dE/go-backend-boilerplate/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestDBSetup represents a temporary PostgreSQL database and its associated resources.
type TestDBSetup struct {
	Pool            *pgxpool.Pool
	TestDBContainer testcontainers.Container
	Config          *config.Config
}

// SetupTestDB creates and configures a PostgreSQL container for integration testing.
// It returns a TestDB struct and a cleanup function to close resources.
func SetupTestDB(t *testing.T) (*TestDBSetup, func()) {
	t.Helper()

	ctx := context.Background()

	// Generate unique DB name for isolation between tests
	databaseName := fmt.Sprintf("test_db_%s", uuid.New().String()[:8])
	databaseUser := "test_user"
	databasePassword := "test_password"

	// Define a container request for Postgres DB
	req := testcontainers.ContainerRequest{
		Image: "postgres:15-alpine",
		Env: map[string]string{
			"POSTGRES_DB":       databaseName,
			"POSTGRES_USER":     databaseUser,
			"POSTGRES_PASSWORD": databasePassword,
		},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForLog("database system is ready to accept connections"),
	}

	// Start the container
	postgresContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "failed to start Postgres container")

	// Get container host and mapped port to connect from the test environment
	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err, "failed to get postgres container host")

	mappedPort, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err, "failed to get postgres container mapped port")
	port := mappedPort.Int()

	// Automatically terminate container after the test finishes
	t.Cleanup(func() {
		err := postgresContainer.Terminate(ctx)
		if err != nil {
			t.Logf("postgres container termination failed %v", err)
		}
	})

	// Build a configuration object similar to production but for tests
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:                  host,
			Port:                  port,
			Name:                  databaseName,
			User:                  databaseUser,
			Password:              databasePassword,
			SSLMode:               "disable",
			MaxOpenConnections:    25,
			MaxIdleConnections:    25,
			ConnectionMaxIdleTime: 300,
			ConnectionMaxLifeTime: 300,
		},
		Primary: config.Primary{
			Env: "test",
		},
		Redis: config.RedisConfig{
			Address: "localhost:6379",
		},
		Integration: config.Integration{
			ResendAPIKey: "test_key",
		},
		Auth: config.AuthConfig{
			SecretKey: "test_secret",
		},
		Server: config.ServerConfig{
			Port:               "8080",
			WriteTimeout:       30,
			ReadTimeout:        30,
			CORSAllowedOrigins: []string{"*"},
		},
	}

	// create logger
	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()

	var db *database.Database
	var lastErr error

	for i := 0; i < 5; i++ {
		time.Sleep(3 * time.Second)
		db, lastErr = database.NewDatabaseConnectionPool(cfg, &logger, nil)
		if lastErr == nil {
			err := db.Pool.Ping(ctx)
			if err == nil {
				break
			} else {
				lastErr = err
				logger.Warn().Err(err).Msg("failed to ping db. Retrying...")
				db.Pool.Close()
			}
		} else {
			logger.Warn().Err(lastErr).Msgf("could not connect to database (%d/5 attempts)", i+1)
		}

	}
	require.NoError(t, lastErr, "database connection failed after multiple attempts")

	// Migrations
	err = database.Migrate(ctx, &logger, cfg)
	require.NoError(t, err, "database migration failed")

	testDBSetup := &TestDBSetup{
		Pool:            db.Pool,
		TestDBContainer: postgresContainer,
		Config:          cfg,
	}

	cleanUp := func() {
		if db.Pool != nil {
			db.Pool.Close()
		}
	}

	return testDBSetup, cleanUp
}


func (db *TestDBSetup) CleanUp(ctx context.Context, logger *zerolog.Logger) error{
	logger.Info().Msg("cleaning up test database...")

	if db.Pool != nil {
		db.Pool.Close()
	}

	if db.TestDBContainer != nil {
		err := db.TestDBContainer.Terminate(ctx)
		if err != nil {
			return fmt.Errorf("failed to terminate postgres test container: %w", err)
		}
	}

	return  nil
}