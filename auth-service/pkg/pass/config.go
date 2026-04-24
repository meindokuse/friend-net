package pass

import "golang.org/x/crypto/bcrypt"

type Config struct {
	Cost int `yaml:"cost" env:"PASS_COST" env-default:"10"`
}

func (c Config) costOrDefault() int {
	if c.Cost <= 0 {
		return bcrypt.DefaultCost
	}

	return c.Cost
}
