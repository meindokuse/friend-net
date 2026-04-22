package pass

import "golang.org/x/crypto/bcrypt"

type Hasher struct {
	cost int
}

func New(cfg Config) *Hasher {
	return &Hasher{cost: cfg.costOrDefault()}
}

func Hash(password string) (string, error) {
	return New(Config{}).Hash(password)
}

func Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func Match(hash, password string) bool {
	return Compare(hash, password) == nil
}

func (h *Hasher) Hash(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}

	return string(hashedPassword), nil
}

func (h *Hasher) Compare(hash, password string) error {
	return Compare(hash, password)
}
