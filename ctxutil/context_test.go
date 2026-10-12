package ctxutil

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSleep(t *testing.T) {
	t.Run("duration elapses", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		const duration = 10 * time.Millisecond
		start := time.Now()
		require.NoError(t, Sleep(ctx, duration))
		// Scheduling may delay the return, but Sleep must not return early.
		assert.GreaterOrEqual(t, time.Since(start), duration)
	})

	t.Run("context interrupts sleep", func(t *testing.T) {
		testContextInterruption(t, func(ctx context.Context) error {
			return Sleep(ctx, time.Hour)
		})
	})
}

func TestWriteChannel(t *testing.T) {
	t.Run("successful send", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		dataCh := make(chan int, 1)

		require.NoError(t, WriteChannel(ctx, dataCh, 42))
		require.Len(t, dataCh, 1)
		assert.Equal(t, 42, <-dataCh)
	})

	t.Run("context interrupts blocked send", func(t *testing.T) {
		// Keep the buffer full so sending cannot compete with cancellation.
		dataCh := make(chan int, 1)
		dataCh <- 7
		testContextInterruption(t, func(ctx context.Context) error {
			return WriteChannel(ctx, dataCh, 42)
		})
		require.Len(t, dataCh, 1)
		assert.Equal(t, 7, <-dataCh)
	})
}

// testContextInterruption checks error propagation and bounds the wait if an
// operation stops responding to cancellation. The result buffer lets the worker
// return even if the test has already failed.
func testContextInterruption(t *testing.T, operation func(context.Context) error) {
	t.Helper()
	for _, mode := range []string{"already canceled", "cancel after start", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			var ctx context.Context
			var cancel context.CancelFunc
			wantErr := context.Canceled
			if mode == "deadline" {
				ctx, cancel = context.WithTimeout(context.Background(), 20*time.Millisecond)
				wantErr = context.DeadlineExceeded
			} else {
				ctx, cancel = context.WithCancel(context.Background())
			}
			defer cancel()
			if mode == "already canceled" {
				cancel()
			}

			started := make(chan struct{})
			result := make(chan error, 1)
			go func() {
				close(started)
				result <- operation(ctx)
			}()
			if mode == "cancel after start" {
				<-started
				// Cancellation may race with entry into the operation; both
				// an already-canceled and a waiting call must return ctx.Err().
				cancel()
			}

			select {
			case err := <-result:
				require.ErrorIs(t, err, wantErr)
			case <-time.After(5 * time.Second):
				t.Fatal("operation did not return after context completion")
			}
		})
	}
}

func TestIsDone(t *testing.T) {
	assert.False(t, IsDone(context.Background()))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	assert.False(t, IsDone(ctx))
	cancel()
	assert.True(t, IsDone(ctx))

	expired, stop := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer stop()
	require.ErrorIs(t, expired.Err(), context.DeadlineExceeded)
	assert.True(t, IsDone(expired))
}
