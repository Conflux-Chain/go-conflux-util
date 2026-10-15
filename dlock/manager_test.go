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

func TestInvalidOwnershipInputs(t *testing.T) {
	ctx := context.Background()
	manager := NewLockManager(fakeBackend{}) // Any backend call would panic.
	for _, tc := range []struct {
		intent *LockIntent
		err    error
	}{
		{nil, ErrInvalidLockIntent},
		{NewLockIntent("key", "", time.Second), ErrEmptyNonce},
	} {
		assert.ErrorIs(t, manager.Acquire(ctx, tc.intent), tc.err)
		assert.ErrorIs(t, manager.Release(ctx, tc.intent), tc.err)
	}

	backend := NewMySQLBackend(nil) // Validation must precede database access.
	ok, err := backend.WriteEntry(ctx, "key", "", time.Second)
	assert.False(t, ok)
	assert.ErrorIs(t, err, ErrEmptyNonce)
	ok, err = backend.DelEntry(ctx, "key", "")
	assert.False(t, ok)
	assert.ErrorIs(t, err, ErrEmptyNonce)
}

func TestInvalidLease(t *testing.T) {
	ctx := context.Background()
	manager := NewLockManager(fakeBackend{})
	backend := NewMySQLBackend(nil)
	for _, lease := range []time.Duration{0, -time.Nanosecond, -time.Second} {
		assert.ErrorIs(t, manager.Acquire(ctx, NewLockIntent("key", "nonce", lease)), ErrInvalidLease)
		ok, err := backend.WriteEntry(ctx, "key", "nonce", lease)
		assert.False(t, ok)
		assert.ErrorIs(t, err, ErrInvalidLease)
	}

	manager = NewLockManager(fakeBackend{
		del: func(context.Context, string, string) (bool, error) { return true, nil },
	})
	assert.NoError(t, manager.Release(ctx, NewLockIntent("key", "nonce", 0)))
}
