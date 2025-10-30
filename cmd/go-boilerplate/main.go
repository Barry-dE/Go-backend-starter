package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
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

const (
	shutdownContextTimeout = 30 * time.Second
	environment            = "development"
)

func main() {

	cfg, err := config.LoadConfig()
	if err != nil {
		panic(fmt.Errorf("failed to load config: %w", err))
	}

	loggerService := logger.NewLoggerService(cfg.Observability)
	defer loggerService.Shutdown()
	log := logger.NewLoggerWithService(cfg.Observability, loggerService)

	if cfg.Primary.Env != environment {
		err := database.Migrate(context.Background(), &log, cfg)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to migrate DB")
		}
	}

	server, err := server.New(cfg, &log, loggerService)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize server")
	}

	repos := repository.NewRepositories(server)

	services, err := service.NewService(server, repos)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize services")
	}

	handlers := handler.NewHandlers(server, services)

	routes := router.NewRouter(server, handlers, services)

	server.ConfigureHTTPServer(routes)

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		err := server.Start()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	<-signalCtx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownContextTimeout)
	defer cancel()

	var once sync.Once

	once.Do(func() {
		err := server.Shutdown(shutdownCtx)
		if err != nil {
			log.Error().Err(err).Msg("graceful shutdown failed")
		} else {
			log.Info().Msg("server exited properly")
		}
	})

}
