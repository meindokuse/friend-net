package mongo

import (
	"strings"

	domainuser "github.com/meindokuse/cloud-drive/user-service/internal/domain/user"
)

// parseDuplicateKey определяет, какое поле вызвало дубликат ключа.
func parseDuplicateKey(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "username"):
		return domainuser.ErrUsernameAlreadyTaken
	case strings.Contains(msg, "email"):
		return domainuser.ErrEmailAlreadyTaken
	case strings.Contains(msg, "phone"):
		return domainuser.ErrPhoneAlreadyTaken
	default:
		return err
	}
}
