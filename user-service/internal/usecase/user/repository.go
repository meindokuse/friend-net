package user

import (
	"context"

	"github.com/google/uuid"

	domainuser "github.com/meindokuse/cloud-drive/user-service/internal/domain/user"
	"github.com/meindokuse/cloud-drive/user-service/internal/domain/shared/vo"
)

// UserRepository — контракт хранилища пользователей.
// Определяется на стороне consumer'а (usecase), реализуется адаптером БД.
//
// Ожидаемые ошибки от реализации:
//   - domainuser.ErrUserNotFound
//   - domainuser.ErrUsernameAlreadyTaken
//   - domainuser.ErrEmailAlreadyTaken
//   - domainuser.ErrPhoneAlreadyTaken
//   - domainuser.ErrVersionConflict
type UserRepository interface {
	Create(ctx context.Context, u *domainuser.User) error
	Update(ctx context.Context, u *domainuser.User) error

	GetByID(ctx context.Context, id uuid.UUID) (*domainuser.User, error)
	GetByUsername(ctx context.Context, username vo.Username) (*domainuser.User, error)
	GetByEmail(ctx context.Context, email vo.Email) (*domainuser.User, error)
	GetByPhone(ctx context.Context, phone vo.Phone) (*domainuser.User, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*domainuser.User, error)

	Search(ctx context.Context, query string, limit, offset int) ([]*domainuser.User, error)
	UpdateLastSeen(ctx context.Context, id uuid.UUID) error
}