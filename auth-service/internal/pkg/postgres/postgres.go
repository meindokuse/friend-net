package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config содержит настройки подключения к PostgreSQL.
type Config struct {
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

// DSN возвращает строку подключения к PostgreSQL.
func (c Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

// NewPool создаёт новый connection pool для PostgreSQL.
func NewPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse pool config: %w", err)
	}

	// Настройки pool
	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	// Таймаут подключения
	connectCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	// Проверяем подключение
	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}

// MustNewPool создаёт pool или паникует при ошибке.
func MustNewPool(ctx context.Context, cfg Config) *pgxpool.Pool {
	pool, err := NewPool(ctx, cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create postgres pool: %s", err))
	}
	return pool
}
