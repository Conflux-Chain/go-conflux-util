package ctxutil

import (
	"context"
	"time"
)

// Sleep waits for d to elapse or ctx to be done. It returns nil when the timer
// is selected, or ctx.Err() when cancellation is selected. If both are ready,
// either may be selected; cancellation does not take priority.
func Sleep(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

// IsDone reports whether ctx.Done() is closed at the time of the check.
// A false result does not guarantee that ctx remains active during subsequent work.
func IsDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// WriteChannel sends data to dataCh or returns ctx.Err() if cancellation is
// selected. A successful send returns nil. If sending and cancellation are both
// ready, either may be selected, even when ctx was canceled before the call.
// Cancellation does not atomically prevent a send.
//
// The caller owns the channel lifecycle and must ensure dataCh is not closed
// before or during the call. Sending to a closed channel can panic even when
// ctx is canceled.
func WriteChannel[T any](ctx context.Context, dataCh chan<- T, data T) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case dataCh <- data:
		return nil
	}
}
