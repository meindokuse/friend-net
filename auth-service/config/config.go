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

// Instance returns singleton config instance
func Instance() *Config {
    once.Do(func() {
        instance = &Config{}

        configPath := os.Getenv("CONFIG_PATH")
        if configPath == "" {
            configPath = "config/config.yaml"
        }

        // Проверяем, существует ли файл
        if _, err := os.Stat(configPath); err == nil {
            fmt.Printf("--- Reading Config File: %s ---\n", configPath)
            // ReadConfig в чистом виде приоритезирует файл, 
            // но если поля в YAML нет, он смотрит на тег env:
            if err := cleanenv.ReadConfig(configPath, instance); err != nil {
                log.Fatalf("config error: %s", err)
            }
        } else {
            fmt.Println("--- Config file not found, reading only from Env ---")
            if err := cleanenv.ReadEnv(instance); err != nil {
                log.Fatalf("env error: %s", err)
            }
        }

        // ВАЖНО: Если ты хочешь, чтобы переменные из .env 
        // ВСЕГДА перекрывали то, что написано в YAML:
        cleanenv.UpdateEnv(instance) 
    })

    return instance
}

// ServerConfig - HTTP server configuration
type ServerConfig struct {
	HTTPAddr        string `yaml:"httpAddr"        env:"HTTP_ADDR"        env-default:":8082"`
	ShutdownTimeout string `yaml:"shutdownTimeout" env:"SHUTDOWN_TIMEOUT" env-default:"10s"`
}

// PostgresConfig - PostgreSQL configuration
type PostgresConfig struct {
	Host            string        `yaml:"host"            env:"POSTGRES_HOST"            env-default:"localhost"`
	Port            int           `yaml:"port"            env:"POSTGRES_PORT"            env-default:"5432"`
	User            string        `yaml:"user"            env:"POSTGRES_USER"            env-default:"postgres"`
	Password        string        `yaml:"password"        env:"POSTGRES_PASSWORD"        env-default:"postgres"`
	Database        string        `yaml:"database"        env:"POSTGRES_DATABASE"        env-default:"auth_db"`
	SSLMode         string        `yaml:"sslMode"         env:"POSTGRES_SSL_MODE"        env-default:"disable"`
	MaxConns        int32         `yaml:"maxConns"        env:"POSTGRES_MAX_CONNS"       env-default:"25"`
	MinConns        int32         `yaml:"minConns"        env:"POSTGRES_MIN_CONNS"       env-default:"5"`
	MaxConnLifetime time.Duration `yaml:"maxConnLifetime" env:"POSTGRES_MAX_CONN_LIFETIME" env-default:"5m"`
	MaxConnIdleTime time.Duration `yaml:"maxConnIdleTime" env:"POSTGRES_MAX_CONN_IDLE_TIME" env-default:"1m"`
	ConnectTimeout  time.Duration `yaml:"connectTimeout"  env:"POSTGRES_CONNECT_TIMEOUT" env-default:"10s"`
}

// DSN returns PostgreSQL connection string
func (c PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

// RedisConfig - Redis configuration
type RedisConfig struct {
	Addr         string        `yaml:"addr"         env:"REDIS_ADDR"         env-default:"localhost:6379"`
	DB           int           `yaml:"db"           env:"REDIS_DB"           env-default:"0"`
	DialTimeout  time.Duration `yaml:"dialTimeout"  env:"REDIS_DIAL_TIMEOUT"  env-default:"5s"`
	ReadTimeout  time.Duration `yaml:"readTimeout"  env:"REDIS_READ_TIMEOUT"  env-default:"3s"`
	WriteTimeout time.Duration `yaml:"writeTimeout" env:"REDIS_WRITE_TIMEOUT" env-default:"3s"`
	PoolSize     int           `yaml:"poolSize"     env:"REDIS_POOL_SIZE"     env-default:"10"`
}

// JWTConfig - JWT configuration
type JWTConfig struct {
	SecretKey     string        `yaml:"secretKey"     env:"JWT_SECRET"         env-required:"true"`
	RefreshSecret string        `yaml:"refreshSecret" env:"JWT_REFRESH_SECRET" env-required:"true"`
	Issuer        string        `yaml:"issuer"        env:"JWT_ISSUER"         env-default:"auth-service"`
	AccessTTL     time.Duration `yaml:"accessTTL"     env-default:"15m"`
	RefreshTTL    time.Duration `yaml:"refreshTTL"    env-default:"720h"`
	GracePeriod   time.Duration `yaml:"gracePeriod"   env-default:"30s"`
}

// PassConfig - Password hashing configuration
type PassConfig struct {
	Cost int `yaml:"cost" env:"PASS_COST" env-default:"10"`
}

// ControllerConfig - HTTP controller configuration
type ControllerConfig struct {
	CookieDomain      string `yaml:"cookieDomain"      env:"COOKIE_DOMAIN"`
	CookieSecure      bool   `yaml:"cookieSecure"      env:"COOKIE_SECURE"       env-default:"false"`
	RefreshCookieName string `yaml:"refreshCookieName" env:"REFRESH_COOKIE_NAME" env-default:"refresh_token"`
}

// GoogleOAuthConfig - Google OAuth configuration
type GoogleOAuthConfig struct {
	ClientID     string   `yaml:"clientID"     env:"GOOGLE_CLIENT_ID"     env-required:"true"`
	ClientSecret string   `yaml:"clientSecret" env:"GOOGLE_CLIENT_SECRET" env-required:"true"`
	RedirectURL  string   `yaml:"redirectURL"  env:"GOOGLE_REDIRECT_URL"`
	Scopes       []string `yaml:"scopes"`
}

// OAuthConfig - OAuth providers configuration
type OAuthConfig struct {
	Google GoogleOAuthConfig `yaml:"google"`
}

// KafkaConfig - Kafka configuration for Outbox Flusher
type KafkaConfig struct {
	Brokers       []string `yaml:"brokers"        env:"KAFKA_BROKERS"        env-default:"localhost:9092" env-separator:","`
	Topic         string   `yaml:"topic"          env:"KAFKA_TOPIC"          env-default:"accounts.events"`
	ConsumerGroup string   `yaml:"consumerGroup" env:"KAFKA_CONSUMER_GROUP" env-default:"auth-service-flusher"`
	Enabled       bool     `yaml:"enabled"        env:"KAFKA_ENABLED"        env-default:"false"`
}

// OutboxConfig - Outbox Flusher configuration
type OutboxConfig struct {
	FlushEnabled  bool          `yaml:"flushEnabled"  env:"OUTBOX_FLUSH_ENABLED"  env-default:"false"`
	FlushInterval time.Duration `yaml:"flushInterval" env:"OUTBOX_FLUSH_INTERVAL" env-default:"5s"`
	BatchSize     int           `yaml:"batchSize"     env:"OUTBOX_BATCH_SIZE"     env-default:"100"`
	LockTimeout   time.Duration `yaml:"lockTimeout"   env:"OUTBOX_LOCK_TIMEOUT"   env-default:"30s"`
	RetryAttempts int           `yaml:"retryAttempts" env:"OUTBOX_RETRY_ATTEMPTS" env-default:"3"`
	RetryDelay    time.Duration `yaml:"retryDelay"    env:"OUTBOX_RETRY_DELAY"    env-default:"1s"`
}

// GracefulConfig - Graceful shutdown configuration
type GracefulConfig struct {
	Timeout time.Duration `yaml:"timeout" env:"GRACEFUL_TIMEOUT" env-default:"10s"`
}

// CORSConfig - CORS configuration
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
	ExposeHeaders    []string
	MaxAge           time.Duration
}

// DefaultCORSConfig returns default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins: []string{
			"http://localhost:3000",
			"http://localhost:5173",
		},
		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"DELETE",
			"PATCH",
			"OPTIONS",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Authorization",
			"Accept",
		},
		AllowCredentials: true,
		ExposeHeaders: []string{
			"Content-Length",
			"Authorization",
		},
		MaxAge: 12 * time.Hour,
	}
}

// Config - main configuration structure
type Config struct {
	Env        string           `yaml:"env"        env:"APP_ENV"        env-default:"local"`
	Server     ServerConfig     `yaml:"server"`
	Controller ControllerConfig `yaml:"controller"`
	Postgres   PostgresConfig   `yaml:"postgres"`
	Redis      RedisConfig      `yaml:"redis"`
	JWT        JWTConfig        `yaml:"jwt"`
	Pass       PassConfig       `yaml:"pass"`
	OAuth      OAuthConfig      `yaml:"oauth"`
	Kafka      KafkaConfig      `yaml:"kafka"`
	Outbox     OutboxConfig     `yaml:"outbox"`
	Graceful   GracefulConfig   `yaml:"graceful"`
}

// ShutdownTimeout returns graceful shutdown timeout
func (c *Config) ShutdownTimeout() time.Duration {
	return c.Graceful.Timeout
}
