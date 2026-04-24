package domain

import "time"

type RegisterInput struct {
	Email    string
	Password string
}

type LoginInput struct {
	Email       string
	Password    string
	Fingerprint string // от клиента
	IP          string // из request
	UserAgent   string // из request
}

type RequestData struct {
	IPAddress   string
	UserAgent   string
	Fingerprint string
}

type RefreshInput struct {
	RefreshToken string // из httpOnly cookie
	Fingerprint  string
}

type LogoutInput struct {
	AccessToken  string
	RefreshToken string
	SessionID    string // если знаем (из access token)
}

type AuthResult struct {
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"` // уйдёт в httpOnly cookie на уровне handler
	TokenType        string    `json:"token_type"`
	ExpiresIn        int64     `json:"expires_in"`
	ExpiresAt        time.Time `json:"expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
	AccountID        string    `json:"account_id"`
}

type AccessTokenInfo struct {
	AccountID string    `json:"account_id"`
	SessionID string    `json:"session_id"`
	JTI       string    `json:"jti"`
	ExpiresAt time.Time `json:"expires_at"`
}

type SessionInfo struct {
	ID         string    `json:"id"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
	Current    bool      `json:"current"`
}
