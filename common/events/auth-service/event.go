package events

import "github.com/google/uuid"

type AccountCreated struct {
  AccountID   uuid.UUID `json:"account_id"`
  Email       string    `json:"email"`
  Username    string    `json:"username"`
  DisplayName string    `json:"display_name"`
  CreatedAt   string    `json:"created_at"` // RFC3339
}

func (e AccountCreated) EventType() string {
  return "account.created"
}