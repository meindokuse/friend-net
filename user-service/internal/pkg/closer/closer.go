package closer

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type Closer struct {
	timeout time.Duration
	funcs   []func(ctx context.Context) error
}

func New(timeout time.Duration) *Closer {
	return &Closer{timeout: timeout}
}

func (c *Closer) Add(fn func(ctx context.Context) error) { c.funcs = append(c.funcs, fn) }

func (c *Closer) CloseAll() {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	var wg sync.WaitGroup
	for _, fn := range c.funcs {
		wg.Add(1)
		go func(f func(context.Context) error) {
			defer wg.Done()
			if err := f(ctx); err != nil {
				slog.Error("close resource failed", "error", err)
			}
		}(fn)
	}
	wg.Wait()
}
