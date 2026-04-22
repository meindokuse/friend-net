package postgresql

import (
	"fmt"
	"net/url"
	"time"
)

type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	Params          map[string]string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
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
