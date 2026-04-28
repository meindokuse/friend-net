package config

import (
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env    string       `yaml:"env"    env:"APP_ENV"    env-default:"local"`
	Server ServerConfig `yaml:"server"`
	Mongo  MongoConfig  `yaml:"mongo"`
	Kafka  KafkaConfig  `yaml:"kafka"`
	Logger LoggerConfig `yaml:"logger"`
}

type ServerConfig struct {
	HTTPAddr        string        `yaml:"httpAddr"        env:"HTTP_ADDR"        env-default:":8081"`
	ShutdownTimeout time.Duration `yaml:"shutdownTimeout" env:"SHUTDOWN_TIMEOUT" env-default:"10s"`
}

type MongoConfig struct {
	URI      string        `yaml:"uri"      env:"MONGO_URI"      env-default:"mongodb://localhost:27017"`
	Database string        `yaml:"database" env:"MONGO_DATABASE" env-default:"user_service"`
	Timeout  time.Duration `yaml:"timeout"  env:"MONGO_TIMEOUT"  env-default:"10s"`
}

type KafkaConfig struct {
	Brokers []string `yaml:"brokers" env:"KAFKA_BROKERS" env-default:"localhost:9092" env-separator:","`
	Topic   string   `yaml:"topic"   env:"KAFKA_TOPIC"   env-default:"accounts.events"`
	GroupID string   `yaml:"groupId" env:"KAFKA_GROUP_ID" env-default:"user-service"`
	Enabled bool     `yaml:"enabled" env:"KAFKA_ENABLED" env-default:"true"`
}

type LoggerConfig struct {
	Level string `yaml:"level" env:"LOG_LEVEL" env-default:"info"`
}

// Load загружает конфиг в следующем порядке приоритета (последний побеждает):
// 1. env-default значения из тегов
// 2. значения из YAML-файла (если существует)
// 3. переменные окружения
func Load() (Config, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./config/config.yaml"
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

// MustLoad - вариант для main, где мы не хотим обрабатывать ошибку.
func MustLoad() Config {
	cfg, err := Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %s", err))
	}
	return cfg
}
