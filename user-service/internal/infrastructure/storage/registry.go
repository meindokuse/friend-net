package storage

import (
	"go.mongodb.org/mongo-driver/mongo"

	userstorage "github.com/meindokuse/cloud-drive/user-service-new/internal/infrastructure/storage/user"
)

type Registry struct {
	User *userstorage.Storage
}

func NewRegistry(db *mongo.Database) (*Registry, error) {
	userStorage, err := userstorage.NewStorage(db)
	if err != nil {
		return nil, err
	}
	return &Registry{User: userStorage}, nil
}
