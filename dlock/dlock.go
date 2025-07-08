package dlock

import (
	"context"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/store"
	"github.com/pkg/errors"
)

var (
	// Custom Errors
	ErrLockNotHeld           = errors.New("lock not held")
	ErrLockAcquisitionFailed = errors.New("failed to acquire lock")
)

// Backend defines the interface for the underlying storage backend of the distributed lock.
type Backend interface {
	// WriteEntry attempts to create or update a lock entry in the storage.
	// It returns true if the lock was obtained, false if not, or an error if something unexpected happened.
	WriteEntry(ctx context.Context, key, nonce string, lease time.Duration) (bool, error)

	// DelEntry attempts to delete a lock entry from the storage.
	// It returns true if the lock was released, false if not, or an error if something unexpected happened.
	DelEntry(ctx context.Context, key, nonce string) (bool, error)
}

// The lock intent to acquire a distributed lock.
type LockIntent struct {
	Key   string        // The unique identifier for the lock.
	Nonce string        // The lock can only be released with the correct nonce.
	Lease time.Duration // The length of the lease duration.
}

func NewLockIntent(key, nonce string, lease time.Duration) *LockIntent {
	return &LockIntent{Key: key, Nonce: nonce, Lease: lease}
}

// LockManager manages distributed locks using a backend storage system.
type LockManager struct {
	backend Backend
}

// NewLockManager creates a new LockManager with the provided backend.
func NewLockManager(be Backend) *LockManager {
	return &LockManager{backend: be}
}

func NewLockManagerFromViper() *LockManager {
	conf := store.MustNewConfigFromViper()
	db := conf.MustOpenOrCreate(&Dlock{})
	return NewLockManager(NewMySQLBackend(db))
}

// Acquire tries to acquire a distributed lock with the given key and lease duration.
// It returns nil error if the lease was successfully obtained.
func (l *LockManager) Acquire(ctx context.Context, li *LockIntent) error {
	ok, err := l.backend.WriteEntry(ctx, li.Key, li.Nonce, li.Lease)
	if err != nil {
		return errors.WithMessage(err, "failed to write lock entry")
	}

	if !ok { // unabled to acquire lock
		return ErrLockAcquisitionFailed
	}

	return nil
}

// Release attempts to release an existing lock.
// It returns nil error if the lock was successfully released.
func (l *LockManager) Release(ctx context.Context, li *LockIntent) error {
	ok, err := l.backend.DelEntry(ctx, li.Key, li.Nonce)
	if err != nil {
		return errors.WithMessage(err, "failed to delete lock entry")
	}

	if !ok { // lock not held
		return ErrLockNotHeld
	}

	return nil
}
