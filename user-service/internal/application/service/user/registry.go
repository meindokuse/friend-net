package user

import (
	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/change_email"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/change_phone"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/create_user"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/delete_user"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/get_user"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/list_users"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/search_users"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/update_last_seen"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/update_profile"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/update_settings"
)

type Registry struct {
	CreateUser     *create_user.Service
	GetUser        *get_user.Service
	UpdateProfile  *update_profile.Service
	UpdateSettings *update_settings.Service
	ChangeEmail    *change_email.Service
	ChangePhone    *change_phone.Service
	DeleteUser     *delete_user.Service
	UpdateLastSeen *update_last_seen.Service
	SearchUsers    *search_users.Service
	ListUsers      *list_users.Service
}
