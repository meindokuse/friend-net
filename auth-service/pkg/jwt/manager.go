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

// ─── Claims ─────────────────────────────────────────

type AccessClaims struct {
	SessionID string `json:"sid"`
	jwtlib.RegisteredClaims
}

// ─── Manager ────────────────────────────────────────

type Manager struct {
	signingKey    []byte
	refreshSecret []byte
	issuer        string
	accessTTL     time.Duration
	refreshTTL    time.Duration
	gracePeriod   time.Duration
}

func NewManager(cfg Config) (*Manager, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("jwt: %w", err)
	}

	return &Manager{
		signingKey:    []byte(cfg.SecretKey),
		refreshSecret: []byte(cfg.RefreshSecret),
		issuer:        cfg.Issuer,
		accessTTL:     cfg.AccessTTL,
		refreshTTL:    cfg.RefreshTTL,
		gracePeriod:   cfg.GracePeriod,
	}, nil
}

// ═══════════════════════════════════════════════════
// ACCESS TOKEN (JWT)
// ═══════════════════════════════════════════════════

// GenerateAccessToken создаёт подписанный JWT
func (m *Manager) GenerateAccessToken(sessionID, userID string) (string, error) {
	now := time.Now().UTC()

	claims := &AccessClaims{
		SessionID: sessionID,
		RegisteredClaims: jwtlib.RegisteredClaims{
			Subject:   userID, // user_id в стандартном поле sub
			Issuer:    m.issuer,
			IssuedAt:  jwtlib.NewNumericDate(now),
			ExpiresAt: jwtlib.NewNumericDate(now.Add(m.accessTTL)),
			ID:        generateJTI(), // уникальный ID для blacklist
		},
	}

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	return token.SignedString(m.signingKey)
}

// VerifyAccessToken проверяет подпись И expiration
// Используется в auth middleware на каждый запрос
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

// ExtractAccessClaims парсит access token БЕЗ проверки expiration
// Проверяет подпись, но игнорирует exp
// Нужно для:
//   - Logout: достать jti для blacklist, даже если token expired
//   - GetSessions: достать session_id из текущего access
func (m *Manager) ExtractAccessClaims(tokenStr string) (sessionID, userID, jti string, expiresAt time.Time, err error) {
	claims := &AccessClaims{}

	_, err = jwtlib.ParseWithClaims(tokenStr, claims, m.keyFunc(), jwtlib.WithoutClaimsValidation())
	if err != nil {
		return "", "", "", time.Time{}, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if claims.SessionID == "" || claims.Subject == "" {
		return "", "", "", time.Time{}, ErrInvalidToken
	}

	exp := time.Time{}
	if claims.ExpiresAt != nil {
		exp = claims.ExpiresAt.Time
	}

	return claims.SessionID, claims.Subject, claims.ID, exp, nil
}

// ═══════════════════════════════════════════════════
// REFRESH TOKEN (Opaque: session_id.random)
// ═══════════════════════════════════════════════════

// GenerateRefreshToken создаёт opaque refresh token
// Формат: {session_id}.{random_base64url}
// session_id нужен чтобы при refresh найти сессию в Redis
// без зависимости от access token
func (m *Manager) GenerateRefreshToken(sessionID string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("jwt: generate refresh: %w", err)
	}

	randomPart := base64.RawURLEncoding.EncodeToString(b)
	return sessionID + "." + randomPart, nil
}

// ParseRefreshToken разбирает refresh token на session_id и random часть
// session_id = UUID (содержит дефисы, но не точки)
// Формат: {uuid}.{base64url_random}
func (m *Manager) ParseRefreshToken(token string) (sessionID, randomPart string, err error) {
	idx := strings.LastIndexByte(token, '.')
	if idx == -1 || idx == 0 || idx == len(token)-1 {
		return "", "", ErrInvalidToken
	}

	sessionID = token[:idx]
	randomPart = token[idx+1:]

	// Базовая валидация
	if len(sessionID) < 36 { // UUID минимум 36 символов
		return "", "", ErrInvalidToken
	}
	if len(randomPart) < 32 { // 32 bytes base64 ≈ 43 символа
		return "", "", ErrInvalidToken
	}

	return sessionID, randomPart, nil
}

// HashRefreshToken создаёт HMAC-SHA256 хеш random части refresh token
// В Redis хранится этот хеш, не plain token
func (m *Manager) HashRefreshToken(randomPart string) string {
	mac := hmac.New(sha256.New, m.refreshSecret)
	mac.Write([]byte(randomPart))
	return hex.EncodeToString(mac.Sum(nil))
}

// ═══════════════════════════════════════════════════
// FINGERPRINT
// ═══════════════════════════════════════════════════

// HashFingerprint создаёт HMAC-SHA256 хеш fingerprint устройства
// Используется отдельный от refresh ключ (signingKey, не refreshSecret)
// чтобы не смешивать домены безопасности
func (m *Manager) HashFingerprint(fingerprint string) string {
	mac := hmac.New(sha256.New, m.signingKey)
	mac.Write([]byte(fingerprint))
	return hex.EncodeToString(mac.Sum(nil))
}

// ═══════════════════════════════════════════════════
// GETTERS
// ═══════════════════════════════════════════════════

func (m *Manager) AccessTTL() time.Duration   { return m.accessTTL }
func (m *Manager) RefreshTTL() time.Duration  { return m.refreshTTL }
func (m *Manager) GracePeriod() time.Duration { return m.gracePeriod }

// ═══════════════════════════════════════════════════
// PRIVATE HELPERS
// ═══════════════════════════════════════════════════

// keyFunc возвращает функцию проверки ключа для jwt.Parse
// Вынесено чтобы не дублировать в каждом методе
func (m *Manager) keyFunc() jwtlib.Keyfunc {
	return func(t *jwtlib.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwtlib.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: unexpected signing method: %v", ErrInvalidToken, t.Header["alg"])
		}
		return m.signingKey, nil
	}
}

// generateJTI создаёт уникальный ID токена для blacklist
func generateJTI() string {
	b := make([]byte, 16)
	rand.Read(b) // crypto/rand — не math/rand
	return hex.EncodeToString(b)
}