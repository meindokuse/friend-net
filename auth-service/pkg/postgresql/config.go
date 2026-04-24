package postgresql

import (
	"fmt"
	"net/url"
	"time"
)

type Config struct {
	Host            string            `yaml:"host"            env:"POSTGRES_HOST"      env-default:"localhost"`
	Port            int               `yaml:"port"            env:"POSTGRES_PORT"      env-default:"5432"`
	User            string            `yaml:"user"            env:"POSTGRES_USER"      env-default:"postgres"`
	Password        string            `yaml:"password"        env:"POSTGRES_PASSWORD"  env-required:"true"`
	Database        string            `yaml:"database"        env:"POSTGRES_DB"        env-required:"true"`
	SSLMode         string            `yaml:"sslMode"         env:"POSTGRES_SSLMODE"   env-default:"disable"`
	Params          map[string]string `yaml:"params"`
	MaxOpenConns    int               `yaml:"maxOpenConns"    env-default:"25"`
	MaxIdleConns    int               `yaml:"maxIdleConns"    env-default:"5"`
	ConnMaxLifetime time.Duration     `yaml:"connMaxLifetime" env-default:"5m"`
}

func (c Config) addr() string {
	host := c.Host
	if host == "" {
		host = "127.0.0.1"
	}

	port := c.Port
	if port == 0 {
		port = 5432
	}

	return fmt.Sprintf("%s:%d", host, port)
}

func (c Config) DSN() string {
	values := url.Values{}
	for key, value := range c.Params {
		values.Set(key, value)
	}

	if values.Get("sslmode") == "" {
		sslMode := c.SSLMode
		if sslMode == "" {
			sslMode = "disable"
		}
		values.Set("sslmode", sslMode)
	}

	return fmt.Sprintf("postgres://%s:%s@%s/%s?%s", c.User, c.Password, c.addr(), c.Database, values.Encode())
}
