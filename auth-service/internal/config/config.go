package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"

	httpcontrollers "github.com/meindokuse/cloud-drive/auth-service/internal/controllers/http"
	googleinfra "github.com/meindokuse/cloud-drive/auth-service/internal/infra"
	"github.com/meindokuse/cloud-drive/auth-service/pkg/jwt"
	"github.com/meindokuse/cloud-drive/auth-service/pkg/pass"
	postgresqlpkg "github.com/meindokuse/cloud-drive/auth-service/pkg/postgresql"
	redispkg "github.com/meindokuse/cloud-drive/auth-service/pkg/redis"
)

type Config struct {
	Env        string                           `yaml:"env"        env:"APP_ENV"        env-default:"local"`
	Server     ServerConfig                     `yaml:"server"`
	Controller httpcontrollers.ControllerConfig `yaml:"controller"`
	Postgres   postgresqlpkg.Config             `yaml:"postgres"`
	Redis      redispkg.Config                  `yaml:"redis"`
	JWT        jwt.Config                       `yaml:"jwt"`
	Pass       pass.Config                      `yaml:"pass"`
	OAuth      OAuthConfig                      `yaml:"oauth"`
}

type ServerConfig struct {
	HTTPAddr        string `yaml:"httpAddr"        env:"HTTP_ADDR"        env-default:":8080"`
	ShutdownTimeout string `yaml:"shutdownTimeout" env:"SHUTDOWN_TIMEOUT" env-default:"10s"`
}

type OAuthConfig struct {
	Google googleinfra.GoogleServiceConfig `yaml:"google"`
}

// Load загружает конфиг в следующем порядке приоритета (последний побеждает):
// 1. env-default значения из тегов
// 2. значения из YAML-файла
// 3. переменные окружения
func Load() (Config, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "../config/config.yaml"
	}

	var cfg Config

	// Если файл существует - читаем YAML + ENV.
	// Если нет (как в k8s, где всё из ENV) - только ENV.
	if _, err := os.Stat(configPath); err == nil {
		if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
			return Config{}, fmt.Errorf("read config file %s: %w", configPath, err)
		}
	} else {
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			return Config{}, fmt.Errorf("read env: %w", err)
		}
	}

	return cfg, nil
}

// MustLoad - вариант для main, где мы не хотим обрабатывать ошибку
func MustLoad() Config {
	cfg, err := Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %s", err))
	}
	return cfg
}