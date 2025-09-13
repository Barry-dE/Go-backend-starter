package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Barry-dE/go-backend-boilerplate/internal/config"
	zerologWriter "github.com/newrelic/go-agent/v3/integrations/logcontext-v2/zerologWriter"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

const ZerologTimeFormat string = "2006-01-02 15:04:05"

type LoggerService struct {
	newRelicApp *newrelic.Application
}

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

	// Enable debug logging only in development environment
	if cfg.Environment == "development" {
		configOptions = append(configOptions, newrelic.ConfigDebugLogger(os.Stdout))
	}

	// Initialize New Relic application
	app, err := newrelic.NewApplication(configOptions...)
	if err != nil {
		fmt.Println("Failed to initialize New Relic:", err)
		return service
	}

	service.newRelicApp = app
	fmt.Println("New Relic initialized successfully.")
	return service
}

// GetNewRelicApp returns the New Relic application instance, if initialized.
func (ls *LoggerService) GetNewRelicApp() *newrelic.Application {
	return ls.newRelicApp
}

// Shutdown gracefully shuts down the New Relic application, if initialized.
func (ls *LoggerService) Shutdown() {
	if ls.newRelicApp != nil {
		ls.newRelicApp.Shutdown(10 * time.Second)
	}
}

// NewLogger creates a zerolog.Logger using basic parameters.
// Useful for simple setups where only log level and environment are needed.
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

func NewLoggerWithConig(cfg *config.MonitoringConfig) zerolog.Logger {
	return NewLoggerWithService(cfg, nil)
}

func NewLoggerWithService(cfg *config.MonitoringConfig, loggerservice *LoggerService) zerolog.Logger {
	var logLevel zerolog.Level
	level := cfg.GetLogLevel()

	switch level {
	case "debug":
		logLevel = zerolog.DebugLevel
	case "info":
		logLevel = zerolog.InfoLevel
	case "warn":
		logLevel = zerolog.WarnLevel
	case "error":
		logLevel = zerolog.ErrorLevel
	
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

// Add distributed tracing metadata from a new relic transaction to logger
func WithTraceContext(logger zerolog.Logger, txn *newrelic.Transaction) zerolog.Logger {
	if txn == nil {
		return logger
	}

	traceMetadata := txn.GetTraceMetadata()
	return logger.With().Str("trace_id", traceMetadata.TraceID).Str("span_id", traceMetadata.SpanID).Logger()
}

// FormatSQLWithArgs reconstructs a SQL query by replacing positional
// placeholders (e.g., $1, $2, …) with the provided argument values.
// This is intended for development use only, as it makes debugging
// and reproducing queries easier by showing the fully interpolated SQL.
func FormatSQLWithArgs(sqlStr string, args []any) string {
	output := sqlStr

	for i, arg := range args {
		placeholder := fmt.Sprintf("$%d", i+1)
		value := fmt.Sprintf("'%v'", arg)
		output = strings.Replace(output, placeholder, value, 1)
	}
	return output
}

// DatabaseLogger creates a zerolog-based logger tailored for database operations.
// It outputs logs to the console with custom formatting:
//   - Long strings are truncated to 200 characters.
//   - JSON byte slices are pretty-printed for readability.
//   - Other values are stringified.
// Each log entry includes a timestamp and a "component=database" field.
// This is useful only in development to inspect SQL queries and parameters
// without overwhelming the logs, to make debugging and query analysis easier.
func DatabaseLogger(level zerolog.Level) zerolog.Logger {
	writer := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: ZerologTimeFormat,
		FormatFieldValue: func(i any) string {
			switch value := i.(type) {
			case string:
				if (len(value)) > 200 {
					return value[:200] + "..."
				}
				return value
			case []byte:
				var obj interface{}
				if err := json.Unmarshal(value, &obj); err == nil {
					prettyPrint, _ := json.MarshalIndent(obj, "", "  ")
					return "\n" + string(prettyPrint)
				}
				return string(value)
			default:
				return fmt.Sprintf("%v", value)
			}

		},
	}
	return zerolog.New(writer).Level(level).With().Timestamp().Str("component", "database").Logger()
}

// GetDBTraceLogLevel maps a zerolog logging level to the integer-based
// trace log levels expected by certain database drivers or tracing tools.
// This allows consistent log severity across both application logs and
// database logs.
// Use this when configuring a database driver that requires numeric
// trace levels instead of zerolog's level types.
func GetDBTraceLogLevel(level zerolog.Level) int {
	switch level {
	case zerolog.DebugLevel:
		return 6
	case zerolog.InfoLevel:
		return 5
	case zerolog.WarnLevel:
		return 4
	case zerolog.ErrorLevel:
		return 2
	default:
		return 0
	}
}

