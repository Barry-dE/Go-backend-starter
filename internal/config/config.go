package config

import (
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	_ "github.com/joho/godotenv/autoload"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
	"github.com/rs/zerolog"
)

type Config struct {
	Primary       Primary           `koanf:"primary" validate:"required"`
	Auth          AuthConfig        `koanf:"auth" validate:"required"`
	Server        ServerConfig      `koanf:"server" validate:"required"`
	Database      DatabaseConfig    `koanf:"database" validate:"required"`
	Redis         RedisConfig       `koanf:"redis" validate:"required"`
	Observability *MonitoringConfig `koanf:"monitoring"`
	Integration   Integration       `koanf:"integration" validate:"required"`
}

type Primary struct {
	Env string `koanf:"env" validate:"required"`
}

type AuthConfig struct {
	SecretKey string `koanf:"secret_key" validate:"required"`
}

type Integration struct {
	ResendAPIKey string `koanf:"resend_api_key" validate:"required"`
}

type ServerConfig struct {
	Port               string   `koanf:"port" validate:"required"`
	ReadTimeout        int      `koanf:"read_timeout" validate:"required"`
	WriteTimeout       int      `koanf:"write_timeout" validate:"required"`
	IdleTimeout        int      `koanf:"idle_timeout" validate:"required"`
	CORSAllowedOrigins []string `koanf:"cors_allowed_origins" validate:"required"`
}

type RedisConfig struct {
	Address string `koanf:"address" validate:"required"`
}

type DatabaseConfig struct {
	Host                  string `koanf:"host" validate:"required"`
	Port                  int    `koanf:"port" validate:"required"`
	Name                  string `koanf:"name" validate:"required"`
	User                  string `koanf:"user" validate:"required"`
	Password              string `koanf:"password"`
	SSLMode               string `koanf:"ssl_mode" validate:"required"`
	MaxOpenConnections    int    `koanf:"max_open_connections" validate:"required"`
	MaxIdleConnections    int    `koanf:"max_idle_connections" validate:"required"`
	ConnectionMaxIdleTime int    `koanf:"connection_max_idle_time" validate:"required"`
	ConnectionMaxLifeTime int    `koanf:"connection_max_life_time" validate:"required"`
}

func LoadConfig() (*Config, error) {

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()

	k := koanf.New(".")

	err := k.Load(env.Provider("BOILERPLATE_", ".", func(s string) string {
		return strings.ToLower(strings.TrimPrefix(s, "BOILERPLATE_"))
	}), nil)

	if err != nil {
		logger.Fatal().Err(err).Msg("There was a problem loading initial environment variables")
	}

	mainConfig := &Config{}

	err = k.Unmarshal("", mainConfig)
	if err != nil {
		logger.Fatal().Err(err).Msg("Could not unmarshal config into struct")
	}

	validate := validator.New()
	err = validate.Struct(mainConfig)
	if err != nil {
		logger.Fatal().Err(err).Msg("Config validation failed")
	}

	// set default monitoring config if not provided
	if mainConfig.Observability == nil {
		mainConfig.Observability = DefaultMonitoringConfig()
	}

	// override service name and environment from primary config
	mainConfig.Observability.ServiceName = "marketmind"
	mainConfig.Observability.Environment = mainConfig.Primary.Env

	// Validate monitoring config
	err = mainConfig.Observability.Validate()
	if err != nil {
		logger.Fatal().Err(err).Msg("Monitoring config validation failed")
	}

	return mainConfig, nil
}
