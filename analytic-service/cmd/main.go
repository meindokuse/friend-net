package main

import (
	"context"

	"github.com/meindokuse/cloud-drive/analytic-service/internal"
)

func main() {
	ctx := context.Background()
	internal.New(ctx).Run(ctx)
}
