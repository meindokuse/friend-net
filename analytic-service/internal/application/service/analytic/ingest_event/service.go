package ingest_event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/meindokuse/cloud-drive/analytic-service/internal/domain/entity"
	"github.com/meindokuse/cloud-drive/analytic-service/internal/pkg/apperr"
)

// Repository enqueues events into the batch buffer; it is intentionally
// non-blocking so Kafka workers are never stalled by ClickHouse back-pressure.
type Repository interface {
	Enqueue(e *entity.Event)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

type Input struct {
	EventID   uuid.UUID
	EventType string
	Service   string
	UserID    *uuid.UUID
	Payload   json.RawMessage
	Timestamp time.Time
}

func (s *Service) Execute(_ context.Context, in Input) error {
	id := in.EventID
	if id == uuid.Nil {
		id = uuid.New()
	}

	payload := ""
	if len(in.Payload) > 0 {
		payload = string(in.Payload)
	}

	e, err := entity.NewEvent(id, in.EventType, in.Service, in.UserID, payload, in.Timestamp)
	if err != nil {
		return fmt.Errorf("%w: %v", apperr.ErrInvalidInput, err)
	}

	s.repo.Enqueue(e)
	return nil
}
