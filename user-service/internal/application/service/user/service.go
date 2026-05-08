package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	authevents "github.com/meindokuse/cloud-drive/common/events/auth-service"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/domain/entity"
	vo "github.com/meindokuse/cloud-drive/user-service/internal/domain/valueobject"
)

type UserRepository interface {
	Create(ctx context.Context, u *entity.User) error
	Update(ctx context.Context, u *entity.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	GetByUsername(ctx context.Context, username vo.Username) (*entity.User, error)
	GetByEmail(ctx context.Context, email vo.Email) (*entity.User, error)
	GetByPhone(ctx context.Context, phone vo.Phone) (*entity.User, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entity.User, error)
	Search(ctx context.Context, query string, limit, offset int) ([]*entity.User, error)
	List(ctx context.Context, params entity.ListParams) ([]*entity.User, entity.PagedUsers, error)
	UpdateLastSeen(ctx context.Context, id uuid.UUID) error
}

type Service struct{ repo UserRepository }

func NewService(repo UserRepository) *Service { return &Service{repo: repo} }

var ErrInvalidInput = errors.New("usecase: invalid input")

type CreateUserInput struct {
	ID          *uuid.UUID
	Username    string
	Email       *string
	Phone       *string
	DisplayName string
}
type UpdateProfileInput struct {
	UserID      uuid.UUID
	DisplayName string
	Bio         *string
	AvatarURL   *string
	Version     int
}
type UpdateSettingsInput struct {
	UserID, Version    uuid.UUID
	WhoCanMessage      string
	WhoCanSeeLastSeen  string
	WhoCanSeeProfile   string
	Language, Timezone string
	V                  int
}

type ChangeEmailInput struct {
	UserID  uuid.UUID
	Email   string
	Version int
}
type ChangePhoneInput struct {
	UserID  uuid.UUID
	Phone   string
	Version int
}
type DeleteUserInput struct {
	UserID  uuid.UUID
	Version int
}
type SearchUsersInput struct {
	Query         string
	Limit, Offset int
}

type PrivacyOutput struct {
	WhoCanMessage     string
	WhoCanSeeLastSeen string
	WhoCanSeeProfile  string
}
type UserOutput struct {
	ID                                     uuid.UUID
	Username                               string
	Email, Phone, Bio, AvatarURL           *string
	DisplayName                            string
	EmailVerified, PhoneVerified, IsActive bool
	Privacy                                PrivacyOutput
	Language, Timezone                     string
	CreatedAt, UpdatedAt                   time.Time
	LastSeenAt                             *time.Time
	Version                                int
}
type PublicUserOutput struct {
	ID          uuid.UUID
	Username    string
	DisplayName string
	Bio         *string
	AvatarURL   *string
	LastSeenAt  *time.Time
}

func (s *Service) CreateUser(ctx context.Context, in CreateUserInput) (*UserOutput, error) {
	username, err := vo.NewUsername(in.Username)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	var emailVO *vo.Email
	if in.Email != nil {
		e, err := vo.NewEmail(*in.Email)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
		}
		emailVO = &e
	}
	var phoneVO *vo.Phone
	if in.Phone != nil {
		p, err := vo.NewPhone(*in.Phone)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
		}
		phoneVO = &p
	}
	id := uuid.New()
	if in.ID != nil {
		id = *in.ID
	}
	u, err := entity.NewUser(id, username, emailVO, phoneVO, in.DisplayName)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	return toUserOutput(u), nil
}

func (s *Service) HandleAccountCreated(ctx context.Context, event *authevents.AccountCreated) error {
	username := strings.TrimSpace(event.Username)
	if username == "" {
		username = strings.Split(event.Email, "@")[0]
	}
	displayName := strings.TrimSpace(event.DisplayName)
	if displayName == "" {
		displayName = username
	}
	email := event.Email
	_, err := s.CreateUser(ctx, CreateUserInput{
		ID:          &event.AccountID,
		Username:    username,
		Email:       &email,
		DisplayName: displayName,
	})
	if err != nil && (errors.Is(err, entity.ErrUsernameAlreadyTaken) || errors.Is(err, entity.ErrEmailAlreadyTaken)) {
		return nil
	}
	return err
}

func (s *Service) GetUserByID(ctx context.Context, id uuid.UUID) (*UserOutput, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toUserOutput(u), nil
}
func (s *Service) GetPublicUserByID(ctx context.Context, id uuid.UUID) (*PublicUserOutput, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toPublicUserOutput(u), nil
}
func (s *Service) GetPublicUserByUsername(ctx context.Context, raw string) (*PublicUserOutput, error) {
	uName, err := vo.NewUsername(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	u, err := s.repo.GetByUsername(ctx, uName)
	if err != nil {
		return nil, err
	}
	return toPublicUserOutput(u), nil
}
func (s *Service) UpdateProfile(ctx context.Context, in UpdateProfileInput) (*UserOutput, error) {
	u, err := s.repo.GetByID(ctx, in.UserID)
	if err != nil {
		return nil, err
	}
	if u.Version() != in.Version {
		return nil, entity.ErrVersionConflict
	}
	if err := u.UpdateProfile(in.DisplayName, in.Bio, in.AvatarURL); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	if err := s.repo.Update(ctx, u); err != nil {
		return nil, err
	}
	return toUserOutput(u), nil
}
func (s *Service) UpdateSettings(ctx context.Context, userID uuid.UUID, in UpdateSettingsInput) (*UserOutput, error) {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u.Version() != in.V {
		return nil, entity.ErrVersionConflict
	}
	settings := entity.Settings{
		Privacy: entity.PrivacySettings{
			WhoCanMessage: entity.PrivacyLevel(in.WhoCanMessage), WhoCanSeeLastSeen: entity.PrivacyLevel(in.WhoCanSeeLastSeen), WhoCanSeeProfile: entity.PrivacyLevel(in.WhoCanSeeProfile),
		}, Language: in.Language, Timezone: in.Timezone,
	}
	if err := u.UpdateSettings(settings); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	if err := s.repo.Update(ctx, u); err != nil {
		return nil, err
	}
	return toUserOutput(u), nil
}
func (s *Service) ChangeEmail(ctx context.Context, in ChangeEmailInput) (*UserOutput, error) {
	e, err := vo.NewEmail(in.Email)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	u, err := s.repo.GetByID(ctx, in.UserID)
	if err != nil {
		return nil, err
	}
	if u.Version() != in.Version {
		return nil, entity.ErrVersionConflict
	}
	existing, err := s.repo.GetByEmail(ctx, e)
	if err != nil && !errors.Is(err, entity.ErrUserNotFound) {
		return nil, err
	}
	if existing != nil && existing.ID() != u.ID() {
		return nil, entity.ErrEmailAlreadyTaken
	}
	u.ChangeEmail(e)
	if err := s.repo.Update(ctx, u); err != nil {
		return nil, err
	}
	return toUserOutput(u), nil
}
func (s *Service) ChangePhone(ctx context.Context, in ChangePhoneInput) (*UserOutput, error) {
	p, err := vo.NewPhone(in.Phone)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	u, err := s.repo.GetByID(ctx, in.UserID)
	if err != nil {
		return nil, err
	}
	if u.Version() != in.Version {
		return nil, entity.ErrVersionConflict
	}
	existing, err := s.repo.GetByPhone(ctx, p)
	if err != nil && !errors.Is(err, entity.ErrUserNotFound) {
		return nil, err
	}
	if existing != nil && existing.ID() != u.ID() {
		return nil, entity.ErrPhoneAlreadyTaken
	}
	u.ChangePhone(p)
	if err := s.repo.Update(ctx, u); err != nil {
		return nil, err
	}
	return toUserOutput(u), nil
}
func (s *Service) DeleteUser(ctx context.Context, in DeleteUserInput) error {
	u, err := s.repo.GetByID(ctx, in.UserID)
	if err != nil {
		return err
	}
	if u.Version() != in.Version {
		return entity.ErrVersionConflict
	}
	if err := u.SoftDelete(); err != nil {
		return err
	}
	return s.repo.Update(ctx, u)
}
func (s *Service) UpdateLastSeen(ctx context.Context, userID uuid.UUID) error {
	return s.repo.UpdateLastSeen(ctx, userID)
}
func (s *Service) SearchUsers(ctx context.Context, in SearchUsersInput) ([]*PublicUserOutput, error) {
	q := strings.TrimSpace(in.Query)
	if q == "" {
		return nil, fmt.Errorf("%w: empty query", ErrInvalidInput)
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := in.Offset
	if offset < 0 {
		offset = 0
	}
	users, err := s.repo.Search(ctx, q, limit, offset)
	if err != nil {
		return nil, err
	}
	return toPublicUserOutputs(users), nil
}
func (s *Service) List(ctx context.Context, params entity.ListParams) ([]*entity.User, entity.PagedUsers, error) {
	return s.repo.List(ctx, params)
}
func (s *Service) GetUsersByIDs(ctx context.Context, ids []uuid.UUID) ([]*PublicUserOutput, error) {
	if len(ids) == 0 {
		return []*PublicUserOutput{}, nil
	}
	if len(ids) > 500 {
		return nil, fmt.Errorf("%w: batch size exceeds 500", ErrInvalidInput)
	}
	users, err := s.repo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return toPublicUserOutputs(users), nil
}

func toUserOutput(u *entity.User) *UserOutput {
	var email, phone *string
	if u.Email() != nil {
		s := u.Email().String()
		email = &s
	}
	if u.Phone() != nil {
		s := u.Phone().String()
		phone = &s
	}
	return &UserOutput{
		ID: u.ID(), Username: u.Username().String(), Email: email, Phone: phone,
		DisplayName: u.Profile().DisplayName, Bio: u.Profile().Bio, AvatarURL: u.Profile().AvatarURL,
		EmailVerified: u.Verification().EmailVerified, PhoneVerified: u.Verification().PhoneVerified,
		Privacy:  PrivacyOutput{WhoCanMessage: string(u.Settings().Privacy.WhoCanMessage), WhoCanSeeLastSeen: string(u.Settings().Privacy.WhoCanSeeLastSeen), WhoCanSeeProfile: string(u.Settings().Privacy.WhoCanSeeProfile)},
		Language: u.Settings().Language, Timezone: u.Settings().Timezone, IsActive: u.IsActive(),
		CreatedAt: u.CreatedAt(), UpdatedAt: u.UpdatedAt(), LastSeenAt: u.LastSeenAt(), Version: u.Version(),
	}
}
func toPublicUserOutput(u *entity.User) *PublicUserOutput {
	return &PublicUserOutput{ID: u.ID(), Username: u.Username().String(), DisplayName: u.Profile().DisplayName, Bio: u.Profile().Bio, AvatarURL: u.Profile().AvatarURL, LastSeenAt: u.LastSeenAt()}
}
func toPublicUserOutputs(users []*entity.User) []*PublicUserOutput {
	out := make([]*PublicUserOutput, 0, len(users))
	for _, u := range users {
		out = append(out, toPublicUserOutput(u))
	}
	return out
}
