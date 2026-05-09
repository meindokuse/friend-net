package account_created

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
)

// Payload is the event payload for account creation.
type Payload struct {
	AccountID   string `json:"account_id"`
	Email       string `json:"email"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
}

// New creates an OutboxEvent for account creation.
func New(accountID uuid.UUID, email, displayName string, createdAt time.Time) (*entity.OutboxEvent, error) {
	payload := Payload{
		AccountID:   accountID.String(),
		Email:       email,
		Username:    usernameFromEmail(email),
		DisplayName: displayName,
		CreatedAt:   createdAt.Format(time.RFC3339),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	return entity.NewOutboxEvent("account", accountID, "account.created", payloadBytes), nil
}

func usernameFromEmail(email string) string {
	for i, c := range email {
		if c == '@' {
			return email[:i]
		}
	}
	return email
}
