package jwt

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("jwt: invalid token")
	ErrTokenExpired = errors.New("jwt: token expired")
)

// AccessClaims represents JWT claims
type AccessClaims struct {
	SessionID string `json:"sid"`
	jwtlib.RegisteredClaims
}

// Manager handles JWT operations
type Manager struct {
	signingKey    []byte
	refreshSecret []byte
	issuer        string
	accessTTL     time.Duration
	refreshTTL    time.Duration
	gracePeriod   time.Duration
}

// NewManager creates a new JWT manager
func NewManager(secretKey, refreshSecret, issuer string, accessTTL, refreshTTL, gracePeriod time.Duration) (*Manager, error) {
	if len(secretKey) < 32 {
		return nil, errors.New("secret key must be at least 32 characters")
	}
	if len(refreshSecret) < 32 {
		return nil, errors.New("refresh secret must be at least 32 characters")
	}

	return &Manager{
		signingKey:    []byte(secretKey),
		refreshSecret: []byte(refreshSecret),
		issuer:        issuer,
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
		gracePeriod:   gracePeriod,
	}, nil
}

// GenerateAccessToken creates a signed JWT
func (m *Manager) GenerateAccessToken(sessionID, userID string) (string, error) {
	now := time.Now().UTC()

	claims := &AccessClaims{
		SessionID: sessionID,
		RegisteredClaims: jwtlib.RegisteredClaims{
			Subject:   userID,
			Issuer:    m.issuer,
			IssuedAt:  jwtlib.NewNumericDate(now),
			ExpiresAt: jwtlib.NewNumericDate(now.Add(m.accessTTL)),
			ID:        generateJTI(),
		},
	}

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	return token.SignedString(m.signingKey)
}

// VerifyAccessToken verifies JWT signature and expiration
func (m *Manager) VerifyAccessToken(tokenStr string) (*AccessClaims, error) {
	claims := &AccessClaims{}

	token, err := jwtlib.ParseWithClaims(tokenStr, claims, m.keyFunc())
	if err != nil {
		if errors.Is(err, jwtlib.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ExtractAccessClaims parses access token without expiration check
func (m *Manager) ExtractAccessClaims(tokenStr string) (sessionID, userID, jti string, expiresAt int64, err error) {
	claims := &AccessClaims{}

	_, err = jwtlib.ParseWithClaims(tokenStr, claims, m.keyFunc(), jwtlib.WithoutClaimsValidation())
	if err != nil {
		return "", "", "", 0, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if claims.SessionID == "" || claims.Subject == "" {
		return "", "", "", 0, ErrInvalidToken
	}

	var exp int64
	if claims.ExpiresAt != nil {
		exp = claims.ExpiresAt.Unix()
	}

	return claims.SessionID, claims.Subject, claims.ID, exp, nil
}

// GenerateRefreshToken creates opaque refresh token
func (m *Manager) GenerateRefreshToken(sessionID string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate refresh: %w", err)
	}

	randomPart := base64.RawURLEncoding.EncodeToString(b)
	return sessionID + "." + randomPart, nil
}

// ParseRefreshToken parses refresh token into session_id and random part
func (m *Manager) ParseRefreshToken(token string) (sessionID, randomPart string, err error) {
	idx := strings.LastIndexByte(token, '.')
	if idx == -1 || idx == 0 || idx == len(token)-1 {
		return "", "", ErrInvalidToken
	}

	sessionID = token[:idx]
	randomPart = token[idx+1:]

	if len(sessionID) < 36 {
		return "", "", ErrInvalidToken
	}
	if len(randomPart) < 32 {
		return "", "", ErrInvalidToken
	}

	return sessionID, randomPart, nil
}

// HashRefreshToken creates HMAC-SHA256 hash of random part
func (m *Manager) HashRefreshToken(randomPart string) string {
	mac := hmac.New(sha256.New, m.refreshSecret)
	mac.Write([]byte(randomPart))
	return hex.EncodeToString(mac.Sum(nil))
}

// HashFingerprint creates HMAC-SHA256 hash of fingerprint
func (m *Manager) HashFingerprint(fingerprint string) string {
	mac := hmac.New(sha256.New, m.signingKey)
	mac.Write([]byte(fingerprint))
	return hex.EncodeToString(mac.Sum(nil))
}

// AccessTTL returns access token TTL
func (m *Manager) AccessTTL() time.Duration {
	return m.accessTTL
}

// RefreshTTL returns refresh token TTL
func (m *Manager) RefreshTTL() time.Duration {
	return m.refreshTTL
}

// GracePeriod returns grace period duration
func (m *Manager) GracePeriod() time.Duration {
	return m.gracePeriod
}

// Now returns current time
func (m *Manager) Now() time.Time {
	return time.Now().UTC()
}

// ExpiresAt returns access token expiry as string
func (m *Manager) ExpiresAt() string {
	return time.Now().UTC().Add(m.accessTTL).Format("2006-01-02T15:04:05Z")
}

// RefreshExpiresAt returns refresh token expiry as string
func (m *Manager) RefreshExpiresAt() string {
	return time.Now().UTC().Add(m.refreshTTL).Format("2006-01-02T15:04:05Z")
}

func (m *Manager) keyFunc() jwtlib.Keyfunc {
	return func(t *jwtlib.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwtlib.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: unexpected signing method: %v", ErrInvalidToken, t.Header["alg"])
		}
		return m.signingKey, nil
	}
}

func generateJTI() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
