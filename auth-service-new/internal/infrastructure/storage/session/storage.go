package session

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
	"github.com/redis/go-redis/v9"
)

const (
	keySession      = "session:"
	keyRefresh      = "refresh:"
	keyUserSessions = "account_sessions:"
	keyBlacklist    = "blacklist:"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrRefreshNotFound = errors.New("refresh pair not found")
)

// Storage implements session repository using Redis
type Storage struct {
	ttl int64
	rdb *redis.Client
}

// NewStorage creates a new session storage
func NewStorage(rdb *redis.Client, ttl int64) *Storage {
	return &Storage{
		ttl: ttl,
		rdb: rdb,
	}
}

// Create creates a new session
func (s *Storage) Create(ctx context.Context, session *entity.Session) error {
	key := keySession + session.ID
	pipe := s.rdb.Pipeline()

	pipe.HSet(ctx, key, map[string]interface{}{
		"account_id":       session.AccountID,
		"status":           string(session.Status),
		"ip_address":       session.IPAddress,
		"user_agent":       session.UserAgent,
		"fingerprint_hash": session.FingerprintHash,
		"created_at":       session.CreatedAt.Unix(),
		"last_seen_at":     session.LastSeenAt.Unix(),
		"expires_at":       session.ExpiresAt.Unix(),
	})
	pipe.Expire(ctx, key, time.Duration(s.ttl)*time.Second)

	// Add to user sessions set
	pipe.SAdd(ctx, keyUserSessions+session.AccountID, session.ID)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// Get retrieves a session by ID
func (s *Storage) Get(ctx context.Context, sessionID string) (*entity.Session, error) {
	key := keySession + sessionID
	data, err := s.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	if len(data) == 0 {
		return nil, ErrSessionNotFound
	}

	return s.parseSession(sessionID, data), nil
}

// GetByAccountID retrieves all sessions for an account
func (s *Storage) GetByAccountID(ctx context.Context, accountID string) ([]*entity.Session, error) {
	sessionIDs, err := s.rdb.SMembers(ctx, keyUserSessions+accountID).Result()
	if err != nil {
		return nil, fmt.Errorf("get session ids: %w", err)
	}

	sessions := make([]*entity.Session, 0, len(sessionIDs))
	for _, sid := range sessionIDs {
		sess, err := s.Get(ctx, sid)
		if err != nil {
			if errors.Is(err, ErrSessionNotFound) {
				// Clean up expired session from set
				s.rdb.SRem(ctx, keyUserSessions+accountID, sid)
				continue
			}
			return nil, err
		}
		sessions = append(sessions, sess)
	}

	return sessions, nil
}

// CountByAccountID counts sessions for an account
func (s *Storage) CountByAccountID(ctx context.Context, accountID string) (int64, error) {
	return s.rdb.SCard(ctx, keyUserSessions+accountID).Result()
}

// UpdateLastSeen updates last seen timestamp
func (s *Storage) UpdateLastSeen(ctx context.Context, sessionID string) error {
	return s.rdb.HSet(ctx, keySession+sessionID, "last_seen_at", time.Now().UTC().Unix()).Err()
}

// Revoke revokes a session
func (s *Storage) Revoke(ctx context.Context, sessionID, accountID string) error {
	pipe := s.rdb.Pipeline()

	pipe.HSet(ctx, keySession+sessionID, "status", string(entity.SessionStatusRevoked))
	pipe.Del(ctx, keyRefresh+sessionID)
	pipe.SRem(ctx, keyUserSessions+accountID, sessionID)

	_, err := pipe.Exec(ctx)
	return err
}

// RevokeAllByAccountID revokes all sessions for an account
func (s *Storage) RevokeAllByAccountID(ctx context.Context, accountID string) error {
	sessionIDs, err := s.rdb.SMembers(ctx, keyUserSessions+accountID).Result()
	if err != nil {
		return fmt.Errorf("get session ids: %w", err)
	}

	if len(sessionIDs) == 0 {
		return nil
	}

	pipe := s.rdb.Pipeline()
	for _, sid := range sessionIDs {
		pipe.HSet(ctx, keySession+sid, "status", string(entity.SessionStatusRevoked))
		pipe.Del(ctx, keyRefresh+sid)
	}
	pipe.Del(ctx, keyUserSessions+accountID)

	_, err = pipe.Exec(ctx)
	return err
}

// SaveRefreshPair saves refresh token pair
func (s *Storage) SaveRefreshPair(ctx context.Context, sessionID string, pair *entity.RefreshPair) error {
	key := keyRefresh + sessionID
	pipe := s.rdb.Pipeline()

	pipe.Del(ctx, key)

	fields := map[string]interface{}{
		"current": pair.Current,
	}
	if pair.Prev != "" {
		fields["prev"] = pair.Prev
		fields["prev_expires_at"] = pair.PrevExpiresAt.Unix()
	}

	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, time.Duration(s.ttl)*time.Second)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("save refresh pair: %w", err)
	}
	return nil
}

// GetRefreshPair retrieves refresh token pair
func (s *Storage) GetRefreshPair(ctx context.Context, sessionID string) (*entity.RefreshPair, error) {
	key := keyRefresh + sessionID
	data, err := s.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("get refresh pair: %w", err)
	}
	if len(data) == 0 {
		return nil, ErrRefreshNotFound
	}

	pair := &entity.RefreshPair{
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

// BlacklistAccessToken blacklists an access token
func (s *Storage) BlacklistAccessToken(ctx context.Context, jti string, ttl int64) error {
	return s.rdb.Set(ctx, keyBlacklist+jti, "1", time.Duration(ttl)*time.Second).Err()
}

// IsBlacklisted checks if access token is blacklisted
func (s *Storage) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	exists, err := s.rdb.Exists(ctx, keyBlacklist+jti).Result()
	if err != nil {
		return false, fmt.Errorf("check blacklist: %w", err)
	}
	return exists > 0, nil
}

func (s *Storage) parseSession(id string, data map[string]string) *entity.Session {
	createdAt, _ := strconv.ParseInt(data["created_at"], 10, 64)
	lastSeenAt, _ := strconv.ParseInt(data["last_seen_at"], 10, 64)
	expiresAt, _ := strconv.ParseInt(data["expires_at"], 10, 64)

	return &entity.Session{
		ID:              id,
		AccountID:       data["account_id"],
		Status:          entity.SessionStatus(data["status"]),
		IPAddress:       data["ip_address"],
		UserAgent:       data["user_agent"],
		FingerprintHash: data["fingerprint_hash"],
		CreatedAt:       time.Unix(createdAt, 0).UTC(),
		LastSeenAt:      time.Unix(lastSeenAt, 0).UTC(),
		ExpiresAt:       time.Unix(expiresAt, 0).UTC(),
	}
}
