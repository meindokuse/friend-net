package outbox

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// OutboxEvent представляет событие в outbox таблице.
type OutboxEvent struct {
	ID            uuid.UUID       `db:"id"`
	AggregateType string          `db:"aggregate_type"`
	AggregateID   uuid.UUID       `db:"aggregate_id"`
	EventType     string          `db:"event_type"`
	Payload       json.RawMessage `db:"payload"`
	CreatedAt     time.Time       `db:"created_at"`
	ProcessedAt   *time.Time      `db:"processed_at"`
}

type internalPayload struct {
	AccountID   uuid.UUID `json:"account_id"`
  	Email       string    `json:"email"`
  	Username    string    `json:"username"`
 	DisplayName string    `json:"display_name"`
  	CreatedAt   string    `json:"created_at"`
}
// NewAccountCreatedEvent создаёт событие для создания аккаунта.
func NewAccountCreatedEvent(accountID uuid.UUID,email, displayName string,createdAt time.Time) (*OutboxEvent,error){
	internalPayload := &internalPayload{
		AccountID: accountID,
		Email: email,
		Username: crateUsernameFromEmail(email),
		DisplayName: displayName,
		CreatedAt: createdAt.Format(time.RFC3339),
	}
	payload,err := json.Marshal(internalPayload)
	if err != nil {
		return nil, fmt.Errorf("outbox NewAccountCreatedEvent: error marshal payload: %w", err)
	}

	return &OutboxEvent{
		ID:            uuid.New(),
		AggregateType: "account",
		AggregateID:   accountID,
		EventType:     "account.created",
		Payload:       payload,
		CreatedAt:     time.Now().UTC(),
	},nil
}

func crateUsernameFromEmail(email string) string {
    atIndex := strings.Index(email, "@")
    if atIndex == -1 {
        return email
    }
    return email[:atIndex]
}
