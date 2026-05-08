package main

import (
	"context"

	"github.com/meindokuse/cloud-drive/auth-service-new/internal"
)

func main() {
	ctx := context.Background()
	internal.New(ctx).Run(ctx)
}
