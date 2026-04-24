package jwt

import (
	"errors"
	"time"
)

var ErrInvalidConfig = errors.New("jwt: invalid config")

type Config struct {
	SecretKey     string        `yaml:"secretKey"     env:"JWT_SECRET"         env-required:"true"`
	RefreshSecret string        `yaml:"refreshSecret" env:"JWT_REFRESH_SECRET" env-required:"true"`
	Issuer        string        `yaml:"issuer"        env:"JWT_ISSUER"         env-default:"auth-service"`
	AccessTTL     time.Duration `yaml:"accessTTL"     env-default:"15m"`
	RefreshTTL    time.Duration `yaml:"refreshTTL"    env-default:"720h"`
	GracePeriod   time.Duration `yaml:"gracePeriod"   env-default:"30s"`
}

func (c Config) Validate() error {
    if len(c.SecretKey) < 32 {
        return errors.New("jwt: secret_key must be at least 32 characters")
    }
    if len(c.RefreshSecret) < 32 {
        return errors.New("jwt: refresh_secret must be at least 32 characters")
    }
    if c.Issuer == "" {
        return errors.New("jwt: issuer is required")
    }
    if c.AccessTTL <= 0 {
        return errors.New("jwt: access_ttl must be positive")
    }
    if c.RefreshTTL <= 0 {
        return errors.New("jwt: refresh_ttl must be positive")
    }
    if c.GracePeriod <= 0 {
        c.GracePeriod = 30 * time.Second
    }
    return nil
}