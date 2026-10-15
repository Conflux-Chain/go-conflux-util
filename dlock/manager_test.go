package dlock

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type fakeBackend struct {
	write func(context.Context, string, string, time.Duration) (bool, error)
	del   func(context.Context, string, string) (bool, error)
}

func (b fakeBackend) WriteEntry(ctx context.Context, key, nonce string, lease time.Duration) (bool, error) {
	return b.write(ctx, key, nonce, lease)
}

func (b fakeBackend) DelEntry(ctx context.Context, key, nonce string) (bool, error) {
	return b.del(ctx, key, nonce)
}

func TestManagerErrorMapping(t *testing.T) {
	storageErr := errors.New("storage unavailable")
	for _, tc := range []struct {
		name         string
		ok           bool
		backendErr   error
		acquireError error
		releaseError error
	}{
		{"success", true, nil, nil, nil},
		{"not held", false, nil, ErrLockAcquisitionFailed, ErrLockNotHeld},
		{"storage error", false, storageErr, storageErr, storageErr},
		{"canceled", false, context.Canceled, context.Canceled, context.Canceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.backendErr == context.Canceled {
				cancel()
			}
			li := NewLockIntent("key", "nonce", time.Second)
			writes, deletes := 0, 0
			manager := NewLockManager(fakeBackend{
				write: func(gotCtx context.Context, key, nonce string, lease time.Duration) (bool, error) {
					writes++
					assert.Equal(t, ctx, gotCtx)
					assert.Equal(t, li.Key, key)
					assert.Equal(t, li.Nonce, nonce)
					assert.Equal(t, li.Lease, lease)
					return tc.ok, tc.backendErr
				},
				del: func(gotCtx context.Context, key, nonce string) (bool, error) {
					deletes++
					assert.Equal(t, ctx, gotCtx)
					assert.Equal(t, li.Key, key)
					assert.Equal(t, li.Nonce, nonce)
					return tc.ok, tc.backendErr
				},
			})
			assert.ErrorIs(t, manager.Acquire(ctx, li), tc.acquireError)
			assert.ErrorIs(t, manager.Release(ctx, li), tc.releaseError)
			assert.Equal(t, 1, writes)
			assert.Equal(t, 1, deletes)
		})
	}
}
