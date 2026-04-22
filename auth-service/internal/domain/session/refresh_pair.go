package domain

import "time"

// RefreshPair — пара хешей refresh токенов для одной сессии
// Хранится отдельно от Session в Redis
type RefreshPair struct {
    Current          string    `json:"current"`                     // HMAC хеш текущего refresh token
    Prev             string    `json:"prev,omitempty"`              // HMAC хеш предыдущего
    PrevExpiresAt    time.Time `json:"prev_expires_at,omitempty"`   // когда prev перестанет быть валидным
}

type RefreshMatchResult int

const (
    RefreshMatchCurrent RefreshMatchResult = iota // совпал с current → обычный rotation
    RefreshMatchPrev                               // совпал с prev → grace period (повтор)
    RefreshMatchNone                                // не совпал → REUSE ATTACK 🚨
)

// Match проверяет к какому хешу относится переданный хеш
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

// Rotate выполняет rotation: current → prev, new → current
func (rp *RefreshPair) Rotate(newHash string, gracePeriod time.Duration) {
    rp.Prev = rp.Current
    rp.PrevExpiresAt = time.Now().UTC().Add(gracePeriod)
    rp.Current = newHash
}