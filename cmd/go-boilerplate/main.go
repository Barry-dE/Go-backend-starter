package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Barry-dE/go-backend-boilerplate/internal/config"
	"github.com/Barry-dE/go-backend-boilerplate/internal/database"
	"github.com/Barry-dE/go-backend-boilerplate/internal/handler"
	"github.com/Barry-dE/go-backend-boilerplate/internal/logger"
	"github.com/Barry-dE/go-backend-boilerplate/internal/repository"
	"github.com/Barry-dE/go-backend-boilerplate/internal/router"
	"github.com/Barry-dE/go-backend-boilerplate/internal/server"
	"github.com/Barry-dE/go-backend-boilerplate/internal/service"
)

const (DefaultContextTimeout = 30
environment = "development")

func main() {
	
    // Load application configuration 
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(fmt.Errorf("failed to load config: %w", err))
	}

	// Initialize New Relic for application monitoring
	loggerService := logger.NewLoggerService(cfg.Observability)
	defer loggerService.Shutdown()

	// Create structured logger with New Relic integration
	log := logger.NewLoggerWithService(cfg.Observability, loggerService)

	// Run database migrations in non-development environments
	if cfg.Primary.Env != environment {
		err := database.Migrate(context.Background(), &log, cfg)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to migrate DB")
		}
	}

	// Initialize server with config, logger, and monitoring
	server, err := server.New(cfg, &log, loggerService)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize server")
	}

	// Initialize data access layer
	repos := repository.NewRepositories(server)

	// Initialize business logic layer 
	services, err := service.NewService(server, repos)
	if err != nil{
		log.Fatal().Err(err).Msg("failed to initialize services")
	}

	// Initialize HTTP handlers 
	handlers := handler.NewHandlers(server, services)

	// Setup routing 
	routes:= router.NewRouter(server, handlers, services )


	// Configure HTTP server
	server.ConfigureHTTPServer(routes)

	// Create cancellable context that listens for interrupt signal
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)

	// Start server in a goroutine so shutdown can be handled gracefully
	go func () {
		err := server.Start()
		if err != nil && !errors.Is(err, http.ErrServerClosed){
			log.Fatal().Err(err).Msg("server error")
		}
	}()
	
	// Block until interrupt signal is received
	<-ctx.Done()

	// Create timeout context for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * DefaultContextTimeout)

	// Attempt graceful shutdown
	if err = server.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("server forced to shutdown")
	}

	// Clean up signal handling and timeout context
	stop()
	cancel()

	log.Info().Msg("server exited properly")
}
