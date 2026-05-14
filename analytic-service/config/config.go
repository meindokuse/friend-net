package config

import (
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
				log.Fatalf("read config file %s: %s", configPath, err)
			}
		} else {
			if err := cleanenv.ReadEnv(instance); err != nil {
				log.Fatalf("read env: %s", err)
			}
		}
	})
	return instance
}

type ServerConfig struct {
	HTTPAddr string `yaml:"httpAddr" env:"HTTP_ADDR" env-default:":8082"`
}

type ClickHouseConfig struct {
	Addrs    []string      `yaml:"addrs" env:"CLICKHOUSE_ADDRS" env-default:"localhost:9000" env-separator:","`
	Database string        `yaml:"database" env:"CLICKHOUSE_DATABASE" env-default:"analytics"`
	Username string        `yaml:"username" env:"CLICKHOUSE_USERNAME" env-default:"default"`
	Password string        `yaml:"password" env:"CLICKHOUSE_PASSWORD" env-default:""`
	Timeout  time.Duration `yaml:"timeout" env:"CLICKHOUSE_TIMEOUT" env-default:"10s"`
}

type KafkaConfig struct {
	Brokers      []string      `yaml:"brokers" env:"KAFKA_BROKERS" env-default:"localhost:9092" env-separator:","`
	Topic        string        `yaml:"topic" env:"KAFKA_TOPIC" env-default:"analytic.event"`
	GroupID      string        `yaml:"groupId" env:"KAFKA_GROUP_ID" env-default:"analytic-service"`
	Enabled      bool          `yaml:"enabled" env:"KAFKA_ENABLED" env-default:"true"`
	MaxWait      time.Duration `yaml:"maxWait" env:"KAFKA_MAX_WAIT" env-default:"500ms"`
	WorkersCount int           `yaml:"workersCount" env:"KAFKA_WORKERS_COUNT" env-default:"8"`
}

type BatcherConfig struct {
	Size          int           `yaml:"size" env:"BATCHER_SIZE" env-default:"500"`
	FlushInterval time.Duration `yaml:"flushInterval" env:"BATCHER_FLUSH_INTERVAL" env-default:"5s"`
	ChannelBuffer int           `yaml:"channelBuffer" env:"BATCHER_CHANNEL_BUFFER" env-default:"10000"`
}

type LoggerConfig struct {
	Level string `yaml:"level" env:"LOG_LEVEL" env-default:"info"`
}

type GracefulConfig struct {
	Timeout time.Duration `yaml:"timeout" env:"GRACEFUL_TIMEOUT" env-default:"15s"`
}

type Config struct {
	Env        string           `yaml:"env" env:"APP_ENV" env-default:"local"`
	Server     ServerConfig     `yaml:"server"`
	ClickHouse ClickHouseConfig `yaml:"clickhouse"`
	Kafka      KafkaConfig      `yaml:"kafka"`
	Batcher    BatcherConfig    `yaml:"batcher"`
	Logger     LoggerConfig     `yaml:"logger"`
	Graceful   GracefulConfig   `yaml:"graceful"`
}
