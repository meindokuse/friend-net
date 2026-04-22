package config

import (
	"os"

	"github.com/goccy/go-yaml"
	"github.com/joho/godotenv"

	httpcontrollers "github.com/meindokuse/cloud-drive/auth-service/internal/controllers/http"
	googleinfra "github.com/meindokuse/cloud-drive/auth-service/internal/infra"
	"github.com/meindokuse/cloud-drive/auth-service/pkg/jwt"
	"github.com/meindokuse/cloud-drive/auth-service/pkg/pass"
	postgresqlpkg "github.com/meindokuse/cloud-drive/auth-service/pkg/postgresql"
	redispkg "github.com/meindokuse/cloud-drive/auth-service/pkg/redis"
)

type Config struct {
	Server     ServerConfig                     `yaml:"server"`
	Controller httpcontrollers.ControllerConfig `yaml:"controller"`
	Postgres   postgresqlpkg.Config             `yaml:"postgres"`
	Redis      redispkg.Config                  `yaml:"redis"`
	JWT        jwt.Config                       `yaml:"jwt"`
	Pass       pass.Config                      `yaml:"pass"`
	OAuth      OAuthConfig                      `yaml:"oauth"`
}

type ServerConfig struct {
	HTTPAddr        string `yaml:"httpAddr"`
	ShutdownTimeout string `yaml:"shutdownTimeout"`
}

type OAuthConfig struct {
	Google googleinfra.GoogleServiceConfig `yaml:"google"`
}

func Load(configPath string) (Config, error) {
	_ = godotenv.Load(".env")

	rawConfig, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(rawConfig, &cfg); err != nil {
		return Config{}, err
	}

	applySecrets(&cfg)

	return cfg, nil
}

func applySecrets(cfg *Config) {
	if cfg == nil {
		return
	}

	cfg.Postgres.Password = envOrDefault("POSTGRES_PASSWORD", cfg.Postgres.Password)
	cfg.Redis.Password = envOrDefault("REDIS_PASSWORD", cfg.Redis.Password)
	cfg.JWT.SecretKey = envOrDefault("JWT_SECRET", cfg.JWT.SecretKey)
	cfg.OAuth.Google.ClientID = envOrDefault("GOOGLE_CLIENT_ID", cfg.OAuth.Google.ClientID)
	cfg.OAuth.Google.ClientSecret = envOrDefault("GOOGLE_CLIENT_SECRET", cfg.OAuth.Google.ClientSecret)
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
