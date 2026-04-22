package redis

import "time"

type Config struct {
	Addr         string
	Username     string
	Password     string
	DB           int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
}

func (c Config) withDefaults() Config {
	if c.Addr == "" {
		c.Addr = "127.0.0.1:6379"
	}

	if c.DialTimeout <= 0 {
		c.DialTimeout = 5 * time.Second
	}

	if c.ReadTimeout <= 0 {
		c.ReadTimeout = 3 * time.Second
	}

	if c.WriteTimeout <= 0 {
		c.WriteTimeout = 3 * time.Second
	}

	return c
}
