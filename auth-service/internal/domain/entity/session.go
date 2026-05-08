package entity

import (
	"time"
)

// SessionStatus represents session status
type SessionStatus string

const (
	SessionStatusActive  SessionStatus = "active"
	SessionStatusRevoked SessionStatus = "revoked"
)

// Session represents a user session entity
type Session struct {
	ID              string
	AccountID       string
	Status          SessionStatus
	IPAddress       string
	UserAgent       string
	FingerprintHash string
	CreatedAt       time.Time
	LastSeenAt      time.Time
	ExpiresAt       time.Time
}

// NewSession creates a new Session entity
func NewSession(id, accountID, fingerprintHash, ip, ua string, ttl time.Duration) *Session {
	now := time.Now().UTC()
	return &Session{
		ID:              id,
		AccountID:       accountID,
		Status:          SessionStatusActive,
		IPAddress:       ip,
		UserAgent:       ua,
		FingerprintHash: fingerprintHash,
		CreatedAt:       now,
		LastSeenAt:      now,
		ExpiresAt:       now.Add(ttl),
	}
}

// IsActive checks if session is active and not expired
func (s *Session) IsActive() bool {
	if s.Status != SessionStatusActive {
		return false
	}
	return time.Now().UTC().Before(s.ExpiresAt)
}

// Revoke marks session as revoked
func (s *Session) Revoke() {
	s.Status = SessionStatusRevoked
}

// UpdateLastSeen updates last seen timestamp
func (s *Session) UpdateLastSeen() {
	s.LastSeenAt = time.Now().UTC()
}
