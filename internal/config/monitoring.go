package config

import (
	"fmt"
	"time"
)

type MonitoringConfig struct {
	ServiceName string            `koanf:"service_name" validate:"required"`
	Environment string            `koanf:"environment" validate:"required"`
	NewRelic    NewRelicConfig    `koanf:"new_relic" validate:"required"`
	Logging     LoggingConfig     `koanf:"logging" validate:"required"`
	HealthCheck HealthCheckConfig `koanf:"health_check" validate:"required"`
}

type NewRelicConfig struct {
	LicenseKey                string `koanf:"license_key" validate:"required"`
	DebugLogging              bool   `koanf:"debug_logging"`
	DistributedTracingEnabled bool   `koanf:"distributed_tracing_enabled"`
	AppLogForwardingEnabled   bool   `koanf:"app_log_forwarding_enabled"`
}

type LoggingConfig struct {
	Level              string        `koanf:"level" validate:"required"`
	SlowQueryThreshold time.Duration `koanf:"slow_query_threshold" `
	Format             string        `koanf:"format" validate:"required"`
}

type HealthCheckConfig struct {
	Enabled  bool          `koanf:"enabled"`
	Interval time.Duration `koanf:"interval" validate:"min=1s"`
	Timeout  time.Duration `koanf:"timeout" validate:"min=1s"`
	Checks   []string      `koanf:"checks"`
}

func DefaultMonitoringConfig() *MonitoringConfig {
	return &MonitoringConfig{
		ServiceName: "marketmind",
		Environment: "development",
		NewRelic: NewRelicConfig{
			LicenseKey:                "",
			DebugLogging:              false,
			DistributedTracingEnabled: true,
			AppLogForwardingEnabled:   true,
		},
		Logging: LoggingConfig{
			Level:              "info",
			SlowQueryThreshold: 200 * time.Millisecond,
			Format:             "json",
		},
		HealthCheck: HealthCheckConfig{
			Enabled:  true,
			Interval: 30 * time.Second,
			Timeout:  5 * time.Second,
			Checks:   []string{"database", "redis", "server"},
		},
	}
}

func (m *MonitoringConfig) Validate() error {
	if m.ServiceName == "" {
		return fmt.Errorf("service_name cannot be empty")
	}

	//Validate log levels
	validLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}

	if !validLevels[m.Logging.Level] {
		return fmt.Errorf("invalid log level: %s (valid levels are debug, info, warn, error)", m.Logging.Level)
	}

	// Validate slow query threshold
	if m.Logging.SlowQueryThreshold < 0 {
		return fmt.Errorf("slow_query_threshold must be non-negative")
	}

	return nil
}

// Get current log level
func (m *MonitoringConfig) GetLogLevel() string {
	switch m.Environment {
	case "production":
		if m.Logging.Level == "" {
			return "info"
		}
	case "development":
		if m.Logging.Level == "" {
			return "debug"
		}
	}
	return m.Logging.Level
}

func (m *MonitoringConfig) IsProductin() bool {
	return m.Environment == "production"
}
