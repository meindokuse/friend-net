package jwt

import (
	"errors"
	"time"
)

var ErrInvalidConfig = errors.New("jwt: invalid config")

type Config struct {
    SecretKey       string        `yaml:"secret_key"`        // мин 32 символа
    RefreshSecret   string        `yaml:"refresh_secret"`    // отдельный секрет для HMAC refresh
    Issuer          string        `yaml:"issuer"`
    AccessTTL       time.Duration `yaml:"access_ttl"`        // 15m
    RefreshTTL      time.Duration `yaml:"refresh_ttl"`       // 30d
    GracePeriod     time.Duration `yaml:"grace_period"`      // 30s
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