package logger

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Barry-dE/go-backend-boilerplate/internal/config"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

const ZerologTimeFormat = "2006-01-02 15:04:05"

// LoggerService wraps New Relic application integration for logging purposes.
type LoggerService struct {
	newRelicApp *newrelic.Application
}

// NewLoggerService initializes New Relic integration if configured in MonitoringConfig.
// Returns a LoggerService, which may or may not have New Relic enabled.
func NewLoggerService(cfg *config.MonitoringConfig) *LoggerService {
	service := &LoggerService{}

	// Skip New Relic setup if no license key is provided
	if cfg.NewRelic.LicenseKey == "" {
		fmt.Println("New Relic license key is not set. Skipping New Relic initialization.")
		return service
	}

	var configOptions []newrelic.ConfigOption
	configOptions = append(configOptions, newrelic.ConfigAppName(cfg.ServiceName),
		newrelic.ConfigLicense(cfg.NewRelic.LicenseKey),
		newrelic.ConfigAppLogForwardingEnabled(cfg.NewRelic.AppLogForwardingEnabled),
		newrelic.ConfigDistributedTracerEnabled(cfg.NewRelic.DistributedTracingEnabled),
	)

	// Enable debug logging only in development to avoid verbose logs and potential sensitive data exposure in production.
	if cfg.Environment == "development" {
		configOptions = append(configOptions, newrelic.ConfigDebugLogger(os.Stdout))
	}

	// Attempt to initialize New Relic with the collected options
	app, err := newrelic.NewApplication(configOptions...)
	if err != nil {
		fmt.Println("Failed to initialize New Relic:", err)
		return service
	}

	service.newRelicApp = app
	fmt.Println("New Relic initialized successfully.")
	return service
}

// GetNewRelicApp exposes the New Relic application instance for advanced integrations.
func (ls *LoggerService) GetNewRelicApp() *newrelic.Application {
	return ls.newRelicApp
}

// Shutdown gracefully shuts down the New Relic application if it was initialized.
func (ls *LoggerService) Shutdown() {
	if ls.newRelicApp != nil {
		ls.newRelicApp.Shutdown(10 * time.Second)
	}
}

// NewLogger creates a zerolog.Logger with the specified log level and environment.
// Uses production or development settings based on isProd.
func NewLogger(level string, isProd bool) zerolog.Logger {
	return NewLoggerWithService(&config.MonitoringConfig{
		Logging: config.LoggingConfig{
			Level: level,
		},

		Environment: func() string {
			if isProd {
				return "production"
			}
			return "development"
		}(),
	}, nil)

}

// NewLoggerWithConig creates a zerolog.Logger using a full MonitoringConfig.
// Useful for advanced or custom logger setups.
func NewLoggerWithConig(cfg *config.MonitoringConfig) zerolog.Logger {
	return NewLoggerWithService(cfg, nil)
}

// NewLoggerWithService creates a zerolog.Logger using MonitoringConfig and an optional LoggerService.
// If in production and New Relic is enabled, logs are forwarded to New Relic.
// In development, logs are printed to the console with stack traces for errors.
func NewLoggerWithService(cfg *config.MonitoringConfig, loggerservice *LoggerService) zerolog.Logger {
	var logLevel zerolog.Level
	level := cfg.GetLogLevel()

	switch level {
	case "debug":
		logLevel = zerolog.DebugLevel
	case "info":
		logLevel = zerolog.InfoLevel
	case "error":
		logLevel = zerolog.ErrorLevel
	case "warn":
		logLevel = zerolog.WarnLevel
	default:
		logLevel = zerolog.InfoLevel
	}

	zerolog.TimeFieldFormat = ZerologTimeFormat
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	var writer io.Writer

	// setup base writer
	var baseWriter io.Writer
	if cfg.IsProductin() && cfg.Logging.Format == "json" {
		// Write to standard output in prod
		baseWriter = os.Stdout

		// Wrap with new Relic zerologwriter for log forwarding in production
		if loggerservice != nil && loggerservice.newRelicApp != nil {
			newRelicWriter := zerologWriter.New(baseWriter, loggerservice.newRelicApp)
			writer = newRelicWriter
		} else {
			writer = baseWriter
		}
	} else {
		// In non-prod  use console writer
		consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: ZerologTimeFormat}
		writer = consoleWriter
	}

	logger := zerolog.New(writer).Level(logLevel).With().Timestamp().Str("service", cfg.ServiceName).Str("environment", cfg.Environment).Logger()

	// Add stack traces for dev errors
	if !cfg.IsProductin() {
		logger = logger.With().Stack().Logger()
	}

	return logger

}

func WithTraceContext(logger zerolog.Logger, txn *newrelic.Transaction) zerolog.Logger {
	if txn == nil {
		return logger
	}

	// get trace metadata
	traceMetadata := txn.GetTraceMetadata()
	return logger.With().Str("trace_id", traceMetadata.TraceID).Str("span_id", traceMetadata.SpanID).Logger()
}
