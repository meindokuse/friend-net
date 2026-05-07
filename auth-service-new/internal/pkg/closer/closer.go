package closer

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"
)

// ErrTimeout is returned if close timeout exceeded
var ErrTimeout = errors.New("timeout waiting for close funcs")

// Closer manages graceful shutdown
type Closer struct {
	mu      sync.Mutex
	once    sync.Once
	done    chan struct{}
	funcs   []func(context.Context) error
	timeout time.Duration
	cancel  context.CancelFunc
}

// New creates a new Closer that listens to OS signals
func New(timeout time.Duration, sig ...os.Signal) *Closer {
	c := &Closer{
		done:    make(chan struct{}),
		timeout: timeout,
	}

	if len(sig) > 0 {
		go func() {
			ch := make(chan os.Signal, 1)
			signal.Notify(ch, sig...)
			<-ch
			signal.Stop(ch)
			if err := c.CloseAll(); err != nil {
				log.Printf("closer: error during CloseAll: %v", err)
			}
		}()
	}

	return c
}

// Add registers close functions
func (c *Closer) Add(funcs ...func(context.Context) error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.funcs = append(c.funcs, funcs...)
}

// Wait waits for CloseAll to complete
func (c *Closer) Wait() {
	<-c.done
}

// CloseAll calls all registered close functions
func (c *Closer) CloseAll() error {
	var retErr error

	c.once.Do(func() {
		defer close(c.done)

		c.mu.Lock()
		funcs := c.funcs
		c.funcs = nil
		c.mu.Unlock()

		ctx := context.Background()
		if c.timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, c.timeout)
			defer cancel()
		}

		for i, f := range funcs {
			err := f(ctx)
			if err != nil {
				log.Printf("closer: error in close func #%d: %v", i+1, err)
				if retErr == nil {
					retErr = err
				}
			}
			if ctx.Err() != nil {
				retErr = ErrTimeout
				break
			}
		}
	})

	return retErr
}
