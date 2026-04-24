package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/session"
	"github.com/redis/go-redis/v9"
)

const (
	keySession      = "session:"          // session:{sid}       → Hash
	keyRefresh      = "refresh:"          // refresh:{sid}       → Hash
	keyUserSessions = "account_sessions:" // account_sessions:{aid} → Set
	keyBlacklist    = "blacklist:"        // blacklist:{jti}     → String
)

// Manager — хранилище сессий в Redis
// Не бизнес-логика, только CRUD + Redis операции
type Manager struct {
	sTTL time.Duration
	rdb  *redis.Client
}

func NewManager(rdb *redis.Client, sTTL time.Duration) *Manager {
	return &Manager{
		sTTL: sTTL,
		rdb:  rdb,
	}
}

// ─── Session CRUD ───────────────────────────────────

func (m *Manager) CreateSession(ctx context.Context, s *domain.Session) error {
	key := keySession + s.ID
	pipe := m.rdb.Pipeline()

	pipe.HSet(ctx, key, map[string]interface{}{
		"account_id":       s.AccountID,
		"status":           string(s.Status),
		"ip_address":       s.IPAddress,
		"user_agent":       s.UserAgent,
		"fingerprint_hash": s.FingerprintHash,
		"created_at":       s.CreatedAt.Unix(),
		"last_seen_at":     s.LastSeenAt.Unix(),
		"expires_at":       s.ExpiresAt.Unix(),
	})
	pipe.Expire(ctx, key, m.sTTL)

	// Добавляем в set сессий пользователя
	pipe.SAdd(ctx, keyUserSessions+s.AccountID, s.ID)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("session: create: %w", err)
	}
	return nil
}

func (m *Manager) GetSession(ctx context.Context, sessionID string) (*domain.Session, error) {
	key := keySession + sessionID
	data, err := m.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("session: get: %w", err)
	}
	if len(data) == 0 {
		return nil, ErrSessionNotFound
	}

	return parseSession(sessionID, data)
}

func (m *Manager) UpdateLastSeen(ctx context.Context, sessionID string) error {
	key := keySession + sessionID
	return m.rdb.HSet(ctx, key,
		"last_seen_at", time.Now().UTC().Unix(),
	).Err()
}

func (m *Manager) RevokeSession(ctx context.Context, sessionID, userID string) error {
	pipe := m.rdb.Pipeline()

	// Помечаем сессию как revoked
	pipe.HSet(ctx, keySession+sessionID, "status", string(domain.SessionStatusRevoked))

	// Удаляем refresh пару — она больше не нужна
	pipe.Del(ctx, keyRefresh+sessionID)

	// Убираем из set пользователя
	pipe.SRem(ctx, keyUserSessions+userID, sessionID)

	_, err := pipe.Exec(ctx)
	return err
}

func (m *Manager) RevokeAllUserSessions(ctx context.Context, userID string) error {
	// Получаем все session ID пользователя
	sessionIDs, err := m.rdb.SMembers(ctx, keyUserSessions+userID).Result()
	if err != nil {
		return fmt.Errorf("session: revoke all: %w", err)
	}

	if len(sessionIDs) == 0 {
		return nil
	}

	pipe := m.rdb.Pipeline()

	for _, sid := range sessionIDs {
		pipe.HSet(ctx, keySession+sid, "status", string(domain.SessionStatusRevoked))
		pipe.Del(ctx, keyRefresh+sid)
	}

	// Очищаем set
	pipe.Del(ctx, keyUserSessions+userID)

	_, err = pipe.Exec(ctx)
	return err
}

func (m *Manager) GetUserSessions(ctx context.Context, userID string) ([]*domain.Session, error) {
	sessionIDs, err := m.rdb.SMembers(ctx, keyUserSessions+userID).Result()
	if err != nil {
		return nil, fmt.Errorf("session: get user sessions: %w", err)
	}

	sessions := make([]*domain.Session, 0, len(sessionIDs))

	for _, sid := range sessionIDs {
		s, err := m.GetSession(ctx, sid)
		if err != nil {
			if errors.Is(err, ErrSessionNotFound) {
				// Сессия протухла по TTL, чистим из set
				m.rdb.SRem(ctx, keyUserSessions+userID, sid)
				continue
			}
			return nil, err
		}

		// Пропускаем revoked
		if s.Status == domain.SessionStatusRevoked {
			continue
		}

		sessions = append(sessions, s)
	}

	return sessions, nil
}

func (m *Manager) CountUserSessions(ctx context.Context, userID string) (int64, error) {
	return m.rdb.SCard(ctx, keyUserSessions+userID).Result()
}

// ─── Refresh Pair CRUD ─────────────────────────────

func (m *Manager) SaveRefreshPair(ctx context.Context, sessionID string, pair *domain.RefreshPair) error {
	key := keyRefresh + sessionID
	pipe := m.rdb.Pipeline()

	// Сначала удаляем старый ключ полностью — чтобы не осталось prev от прошлого
	pipe.Del(ctx, key)

	fields := map[string]interface{}{
		"current": pair.Current,
	}
	if pair.Prev != "" {
		fields["prev"] = pair.Prev
		fields["prev_expires_at"] = pair.PrevExpiresAt.Unix()
	}

	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, m.sTTL)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("session: save refresh pair: %w", err)
	}
	return nil
}

func (m *Manager) GetRefreshPair(ctx context.Context, sessionID string) (*domain.RefreshPair, error) {
	key := keyRefresh + sessionID
	data, err := m.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("session: get refresh pair: %w", err)
	}
	if len(data) == 0 {
		return nil, ErrRefreshNotFound
	}

	return parseRefreshPair(data)
}

func (m *Manager) DeleteRefreshPair(ctx context.Context, sessionID string) error {
	return m.rdb.Del(ctx, keyRefresh+sessionID).Err()
}

func (m *Manager) DeleteSession(ctx context.Context, sessionID string) error {
	err := m.rdb.Del(ctx, keySession+sessionID).Err()
	if err != nil {
		return fmt.Errorf("session: delete user session: %w", err)
	}
	return nil
}

// ─── Access Token Blacklist ────────────────────────

func (m *Manager) BlacklistAccessToken(ctx context.Context, jti string, ttl time.Duration) error {
	return m.rdb.Set(ctx, keyBlacklist+jti, "1", ttl).Err()
}

func (m *Manager) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	exists, err := m.rdb.Exists(ctx, keyBlacklist+jti).Result()
	if err != nil {
		return false, fmt.Errorf("session: check blacklist: %w", err)
	}
	return exists > 0, nil
}

// ─── Parsers ───────────────────────────────────────

func parseSession(id string, data map[string]string) (*domain.Session, error) {
	createdAt, _ := strconv.ParseInt(data["created_at"], 10, 64)
	lastSeenAt, _ := strconv.ParseInt(data["last_seen_at"], 10, 64)
	expiresAt, _ := strconv.ParseInt(data["expires_at"], 10, 64)

	return &domain.Session{
		ID:              id,
		AccountID:       data["account_id"],
		Status:          domain.SessionStatus(data["status"]),
		IPAddress:       data["ip_address"],
		UserAgent:       data["user_agent"],
		FingerprintHash: data["fingerprint_hash"],
		CreatedAt:       time.Unix(createdAt, 0).UTC(),
		LastSeenAt:      time.Unix(lastSeenAt, 0).UTC(),
		ExpiresAt:       time.Unix(expiresAt, 0).UTC(),
	}, nil
}

func parseRefreshPair(data map[string]string) (*domain.RefreshPair, error) {
	pair := &domain.RefreshPair{
		Current: data["current"],
	}

	if prev, ok := data["prev"]; ok && prev != "" {
		pair.Prev = prev
		if ts, ok := data["prev_expires_at"]; ok {
			unix, _ := strconv.ParseInt(ts, 10, 64)
			pair.PrevExpiresAt = time.Unix(unix, 0).UTC()
		}
	}

	return pair, nil
}
