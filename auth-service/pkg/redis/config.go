package redis

import "time"


type Config struct {
	Addr         string        `yaml:"addr"         env:"REDIS_ADDR"      env-default:"localhost:6379"`
	Username     string        `yaml:"username"     env:"REDIS_USERNAME"`
	Password     string        `yaml:"password"     env:"REDIS_PASSWORD"`
	DB           int           `yaml:"db"           env:"REDIS_DB"        env-default:"0"`
	DialTimeout  time.Duration `yaml:"dialTimeout"  env-default:"5s"`
	ReadTimeout  time.Duration `yaml:"readTimeout"  env-default:"3s"`
	WriteTimeout time.Duration `yaml:"writeTimeout" env-default:"3s"`
	PoolSize     int           `yaml:"poolSize"     env-default:"10"`
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
