package pass

import "golang.org/x/crypto/bcrypt"

// Hasher handles password hashing
type Hasher struct {
	cost int
}

// New creates a new Hasher
func New(cost int) *Hasher {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}
	return &Hasher{cost: cost}
}

// Hash hashes a password
func (h *Hasher) Hash(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// Compare compares hash and password
func (h *Hasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// Match checks if hash matches password
func (h *Hasher) Match(hash, password string) bool {
	return h.Compare(hash, password) == nil
}
