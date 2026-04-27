package consumer

import (
	"time"

	"github.com/google/uuid"
)

type AccoutCreateEvent struct {
	ID 			uuid.UUID
	AggregateID uuid.UUID
	Email 		string
	Username	string
	FirstName	string
	SecondName  string
	CreatedAt   time.Time
}