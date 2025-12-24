package ctxutil

import (
	"context"
	"time"
)

// Sleep waits for the duration to elapse. If the given context done, it will return error ctx.Err().
func Sleep(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

// IsDone returns whether the give context is done.
func IsDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// WriteChannel writes the given data to channel. If the context is done, it will return error ctx.Err().
func WriteChannel[T any](ctx context.Context, dataCh chan<- T, data T) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case dataCh <- data:
		return nil
	}
}
