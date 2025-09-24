// Package database provides a PostgreSQL connection pool configured with
// logging (zerolog) and optional New Relic instrumentation. It ensures that
// a connection can be established before returning the pool.

package database

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/Barry-dE/go-backend-boilerplate/internal/config"
	loggerConfig "github.com/Barry-dE/go-backend-boilerplate/internal/logger"
	pgxZeroLog "github.com/jackc/pgx-zerolog"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/newrelic/go-agent/v3/integrations/nrpgx5"
	"github.com/rs/zerolog"
)

type Database struct {
	Pool *pgxpool.Pool
	log  *zerolog.Logger
}

const DatabasePingTimeout = 10

type multiEnvironmentTracer struct {
	tracers []any
}

// TraceQueryStart is called by pgx when a query begins execution.
// It forwards the TraceQueryStart event to all tracers stored in multiEnvironmentTracer.
// Each tracer may return a new context (e.g., attaching trace IDs or metadata).
// We loop through all registered tracers, updating the context as we go.
func (met *multiEnvironmentTracer) TraceQueryStart(ctx context.Context, connection *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	for _, tracer := range met.tracers {
		if t, ok := tracer.(interface {
			TraceQueryStart(context.Context, *pgx.Conn, pgx.TraceQueryStartData) context.Context
		}); ok {
			ctx = t.TraceQueryStart(ctx, connection, data)
		}
	}
	return ctx
}

// TraceQueryEnd is called by pgx after a query has finished executing.
// It forwards the TraceQueryEnd event (with query results, timings, or errors)
// to every tracer in multiEnvironmentTracer. Unlike TraceQueryStart,
// this does not return a new context — it just notifies tracers that the query ended.
func (met *multiEnvironmentTracer) TraceQueryEnd(ctx context.Context, connection *pgx.Conn, data pgx.TraceQueryEndData) {
	for _, tracer := range met.tracers {
		if t, ok := tracer.(interface {
			TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData)
		}); ok {
			t.TraceQueryEnd(ctx, connection, data)
		}
	}

}

func NewDatabaseConnectionPool(cfg *config.Config, logger *zerolog.Logger, loggerService *loggerConfig.LoggerService) (*Database, error) {
	hostPort := net.JoinHostPort(cfg.Database.Host, strconv.Itoa(cfg.Database.Port))

	// URL-encode the database password
	encodePassword := url.QueryEscape(cfg.Database.Password)
	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s", cfg.Database.User, encodePassword, hostPort, cfg.Database.Name, cfg.Database.SSLMode)

	// parse dsn to create a pool of connections
	pgxPoolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pgx pool config: %w", err)
	}

	// Instrument database with new relic
	if loggerService != nil && loggerService.GetNewRelicApp() != nil {
		pgxPoolConfig.ConnConfig.Tracer = nrpgx5.NewTracer()
	}

	if cfg.Primary.Env == "local" {
		globalLogLevel := logger.GetLevel()
		pgxLogger := loggerConfig.DatabaseLogger(globalLogLevel)

		// chain traces, new relic first,then local logging
		if pgxPoolConfig.ConnConfig.Tracer != nil {
			// if new relic tracer exist, create a multi tracer
			devTracer := &tracelog.TraceLog{
				Logger:   pgxZeroLog.NewLogger(pgxLogger),
				LogLevel: tracelog.LogLevel(loggerConfig.GetDBTraceLogLevel(globalLogLevel)),
			}

			pgxPoolConfig.ConnConfig.Tracer = &multiEnvironmentTracer{
				tracers: []any{pgxPoolConfig.ConnConfig.Tracer, devTracer},
			}
		} else {
			pgxPoolConfig.ConnConfig.Tracer = &tracelog.TraceLog{
				Logger:   pgxZeroLog.NewLogger(pgxLogger),
				LogLevel: tracelog.LogLevel(loggerConfig.GetDBTraceLogLevel(globalLogLevel)),
			}
		}

	}

	pool, err := pgxpool.NewWithConfig(context.Background(), pgxPoolConfig)
	if err != nil {
		return nil, fmt.Errorf("pool creation failed: %w", err)
	}

	database := &Database{
		Pool: pool,
		log:  logger,
	}

	ctx, cancel := context.WithTimeout(context.Background(), DatabasePingTimeout*time.Second)
	defer cancel()
	if err = pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("Database ping failed: %w", err)
	}

	logger.Info().Msg("Database connected successfully")

	return database, nil
}

// Close gracefully shuts down the database connection pool.
func (db *Database) Close() error {
	db.log.Info().Msg("Closing database connection pool")
	db.Pool.Close()
	return nil
}
