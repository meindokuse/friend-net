package storage

import (
	"go.mongodb.org/mongo-driver/mongo"

	idempotencystorage "github.com/meindokuse/cloud-drive/user-service-new/internal/infrastructure/storage/idempotency"
	userstorage "github.com/meindokuse/cloud-drive/user-service-new/internal/infrastructure/storage/user"
)

type Registry struct {
	User        *userstorage.Storage
	Idempotency *idempotencystorage.Storage
}

func NewRegistry(db *mongo.Database) (*Registry, error) {
	userStorage, err := userstorage.NewStorage(db)
	if err != nil {
		return nil, err
	}
	idempotencyStorage, err := idempotencystorage.NewStorage(db)
	if err != nil {
		return nil, err
	}
	return &Registry{User: userStorage, Idempotency: idempotencyStorage}, nil
}
