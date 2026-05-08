package errors

import "github.com/meindokuse/cloud-drive/user-service-new/internal/domain/entity"

var (
	ErrEmailOrPhoneRequired = entity.ErrEmailOrPhoneRequired
	ErrDisplayNameRequired  = entity.ErrDisplayNameRequired
	ErrDisplayNameTooLong   = entity.ErrDisplayNameTooLong
	ErrBioTooLong           = entity.ErrBioTooLong
	ErrInvalidPrivacyLevel  = entity.ErrInvalidPrivacyLevel
	ErrAlreadyDeleted       = entity.ErrAlreadyDeleted

	ErrUserNotFound         = entity.ErrUserNotFound
	ErrUsernameAlreadyTaken = entity.ErrUsernameAlreadyTaken
	ErrEmailAlreadyTaken    = entity.ErrEmailAlreadyTaken
	ErrPhoneAlreadyTaken    = entity.ErrPhoneAlreadyTaken
	ErrVersionConflict      = entity.ErrVersionConflict
)
