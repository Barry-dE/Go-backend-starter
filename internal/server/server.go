package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Barry-dE/go-backend-boilerplate/internal/config"
	"github.com/Barry-dE/go-backend-boilerplate/internal/database"
	"github.com/Barry-dE/go-backend-boilerplate/internal/lib/job"
	loggerPackage "github.com/Barry-dE/go-backend-boilerplate/internal/logger"
	newRelicRedis "github.com/newrelic/go-agent/v3/integrations/nrredis-v9"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// Server holds all dependencies and services used by the application.
type Server struct {
	Config        *config.Config
	DB            *database.Database
	Logger        *zerolog.Logger
	LoggerService *loggerPackage.LoggerService
	Redis         *redis.Client
	httpServer    *http.Server
	Job           *job.JobService
}

// New creates and initializes a new Server instance.
func New(cfg *config.Config, logger *zerolog.Logger, loggerService *loggerPackage.LoggerService) (*Server, error) {

	// Initialize the database connection pool.
	db, err := database.NewDatabaseConnectionPool(cfg, logger, loggerService)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database %w", err)
	}

	// Initialize the Redis client using configuration details.
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.Redis.Address,
	})

	// Attach New Relic monitoring to Redis if available.
	if loggerService != nil && loggerService.GetNewRelicApp() != nil {
		redisClient.AddHook(newRelicRedis.NewHook(redisClient.Options()))
	}

	// Test the Redis connection, but don't block startup if it's unavailable.
	ctx, cancle := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancle()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error().Err(err).Msg("Failed to connect to Redis, continuing without Redis")
	}

	// Initialize the background job service.
	jobService := job.NewJobService(logger, cfg)
	jobService.InitHandlers(cfg, logger)

	// Start the job service and return an error if it fails.
	if err := jobService.Start(); err != nil {
		return nil, err
	}

	// Assemble the server with all initialized components.
	server := &Server{
		Config:        cfg,
		DB:            db,
		Logger:        logger,
		LoggerService: loggerService,
		Redis:         redisClient,
		Job:           jobService,
	}

	return server, nil
}


// ConfigureHTTPServer sets up the HTTP server with the provided handler and configuration values.
// It applies timeouts and port settings from the server configuration.
func (s *Server) ConfigureHTTPServer(handler http.Handler) {
	s.httpServer = &http.Server{
		Addr:         ":" + s.Config.Server.Port,
		Handler:      handler,
		ReadTimeout:  time.Duration(s.Config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.Config.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(s.Config.Server.IdleTimeout),
	}
}

// Start launches the HTTP server and begins listening for incoming requests.
func (s *Server) Start() error {
	if s.httpServer == nil {
		return errors.New(("http server not configured, call ConfigureHTTPServer first"))
	}

	// Log that the server is starting, including environment and port info.
	s.Logger.Info().Str("port", s.Config.Server.Port).Str("env", s.Config.Primary.Env).Msg("Starting HTTP server")

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the server and cleans up resources.
func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown http server: %w", err)
	}

	if err := s.DB.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	// Stop any running background jobs if present.
	if s.Job != nil {
		s.Job.Stop()
	}

	return nil
}
