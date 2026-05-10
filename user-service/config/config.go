package config

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

var (
	instance *Config
	once     sync.Once
)

func Instance() *Config {
	once.Do(func() {
		instance = &Config{}
		configPath := os.Getenv("CONFIG_PATH")
		if configPath == "" {
			configPath = "config/config.yaml"
		}
		if _, err := os.Stat(configPath); err == nil {
			if err := cleanenv.ReadConfig(configPath, instance); err != nil {
				log.Fatalf("read config file %s error: %s", configPath, err.Error())
			}
		} else {
			if err := cleanenv.ReadEnv(instance); err != nil {
				log.Fatalf("read env error: %s", err.Error())
			}
		}
	})
	return instance
}

type ServerConfig struct {
	HTTPAddr        string        `yaml:"httpAddr" env:"HTTP_ADDR" env-default:":8081"`
	ShutdownTimeout time.Duration `yaml:"shutdownTimeout" env:"SHUTDOWN_TIMEOUT" env-default:"10s"`
}

type MongoConfig struct {
	URI      string        `yaml:"uri" env:"MONGO_URI" env-default:"mongodb://localhost:27017"`
	Database string        `yaml:"database" env:"MONGO_DATABASE" env-default:"user_service"`
	Timeout  time.Duration `yaml:"timeout" env:"MONGO_TIMEOUT" env-default:"10s"`
}

type KafkaConfig struct {
	Brokers      []string      `yaml:"brokers" env:"KAFKA_BROKERS" env-default:"localhost:9092" env-separator:","`
	Topic        string        `yaml:"topic" env:"KAFKA_TOPIC" env-default:"accounts.events"`
	GroupID      string        `yaml:"groupId" env:"KAFKA_GROUP_ID" env-default:"user-service"`
	Enabled      bool          `yaml:"enabled" env:"KAFKA_ENABLED" env-default:"true"`
	MaxWait      time.Duration `yaml:"maxWait" env:"KAFKA_MAX_WAIT" env-default:"500ms"`
	WorkersCount int           `yaml:"workersCount" env:"KAFKA_WORKERS_COUNT" env-default:"16"`
	MaxRetries   int           `yaml:"maxRetries" env:"KAFKA_MAX_RETRIES" env-default:"3"`
	MaxDLQRetries int          `yaml:"maxDlqRetries" env:"KAFKA_MAX_DLQ_RETRIES" env-default:"3"`
}

type GracefulConfig struct {
	Timeout time.Duration `yaml:"timeout" env:"GRACEFUL_TIMEOUT" env-default:"10s"`
}

type LoggerConfig struct {
	Level string `yaml:"level" env:"LOG_LEVEL" env-default:"info"`
}

type Config struct {
	Env      string         `yaml:"env" env:"APP_ENV" env-default:"local"`
	Server   ServerConfig   `yaml:"server"`
	Mongo    MongoConfig    `yaml:"mongo"`
	Kafka    KafkaConfig    `yaml:"kafka"`
	Logger   LoggerConfig   `yaml:"logger"`
	Graceful GracefulConfig `yaml:"graceful"`
}

func (c MongoConfig) DSN() string {
	return fmt.Sprintf("%s/%s", c.URI, c.Database)
}
