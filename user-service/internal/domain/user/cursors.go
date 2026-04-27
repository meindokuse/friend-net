package user

import "github.com/google/uuid"

type UsernameCursor struct {
	Username string
	ID       uuid.UUID
}

type ListParams struct {
	Limit  int
	Cursor *UsernameCursor
}

type PagedUsers struct {
	Items     []*User
	NextCursor UsernameCursor
	HasMore   bool
}