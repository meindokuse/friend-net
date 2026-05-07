package processor

import (
	"context"

	authevents "github.com/meindokuse/cloud-drive/common/events/auth-service"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user"
)

type AccountCreatedProcessor struct {
	service *user.Service
}

func NewAccountCreatedProcessor(service *user.Service) *AccountCreatedProcessor {
	return &AccountCreatedProcessor{service: service}
}

func (p *AccountCreatedProcessor) HandleAccountCreated(ctx context.Context, event *authevents.AccountCreated) error {
	return p.service.HandleAccountCreated(ctx, event)
}
