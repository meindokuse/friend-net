package domain

import (
	"time"
)

type SessionStatus string

const (
	SessionStatusActive  SessionStatus = "active"
	SessionStatusRevoked SessionStatus = "revoked"
)

type Session struct {
	ID              string        `json:"id"`
	AccountID       string        `json:"account_id"`
	Status          SessionStatus `json:"status"`
	IPAddress       string        `json:"ip_address"`
	UserAgent       string        `json:"user_agent"`
	FingerprintHash string        `json:"fingerprint_hash"`
	CreatedAt       time.Time     `json:"created_at"`
	LastSeenAt      time.Time     `json:"last_seen_at"`
	ExpiresAt       time.Time     `json:"expires_at"`
}

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

func (s *Session) IsActive() bool {
	if s.Status != SessionStatusActive {
		return false
	}
	return time.Now().UTC().Before(s.ExpiresAt)
}

func (s *Session) Revoke() {
	s.Status = SessionStatusRevoked
}
