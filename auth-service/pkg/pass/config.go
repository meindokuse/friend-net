package pass

import "golang.org/x/crypto/bcrypt"

type Config struct {
	Cost int
}

func (c Config) costOrDefault() int {
	if c.Cost <= 0 {
		return bcrypt.DefaultCost
	}

	return c.Cost
}
