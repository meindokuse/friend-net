package event

import "context"

type contextKey struct{}

// WithContext attaches a new Buffer (backed by flusher) to ctx and returns both.
func WithContext(ctx context.Context, flusher Flusher) (context.Context, *Buffer) {
	buf := NewBuffer(flusher)
	return context.WithValue(ctx, contextKey{}, buf), buf
}

// Add appends an event to the Buffer stored in ctx (no-op if none).
func Add(ctx context.Context, e *Event) {
	if buf, ok := ctx.Value(contextKey{}).(*Buffer); ok {
		buf.Add(e)
	}
}

// FromContext retrieves the Buffer from ctx.
func FromContext(ctx context.Context) (*Buffer, bool) {
	buf, ok := ctx.Value(contextKey{}).(*Buffer)
	return buf, ok
}
