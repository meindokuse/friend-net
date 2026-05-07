package entity

import "time"

// RefreshPair represents a pair of refresh token hashes for a session
type RefreshPair struct {
	Current       string    `json:"current"`                   // HMAC hash of current refresh token
	Prev          string    `json:"prev,omitempty"`            // HMAC hash of previous
	PrevExpiresAt time.Time `json:"prev_expires_at,omitempty"` // When prev stops being valid
}

// RefreshMatchResult represents the result of refresh token matching
type RefreshMatchResult int

const (
	RefreshMatchCurrent RefreshMatchResult = iota // Matched current - normal rotation
	RefreshMatchPrev                              // Matched prev - grace period (retry)
	RefreshMatchNone                              // No match - REUSE ATTACK
)

// Match checks which hash the provided hash belongs to
func (rp *RefreshPair) Match(hash string) RefreshMatchResult {
	if hash == rp.Current {
		return RefreshMatchCurrent
	}

	if rp.Prev != "" && hash == rp.Prev {
		if time.Now().UTC().Before(rp.PrevExpiresAt) {
			return RefreshMatchPrev
		}
	}

	return RefreshMatchNone
}

// Rotate performs rotation: current -> prev, new -> current
func (rp *RefreshPair) Rotate(newHash string, gracePeriod time.Duration) {
	rp.Prev = rp.Current
	rp.PrevExpiresAt = time.Now().UTC().Add(gracePeriod)
	rp.Current = newHash
}

// SetCurrent sets a new current hash (for grace period rotation)
func (rp *RefreshPair) SetCurrent(newHash string) {
	rp.Current = newHash
}
