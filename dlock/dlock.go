package dlock

import (
	"context"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/store"
	"github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	// Custom Errors
	ErrLockNotHeld           = errors.New("lock not held")
	ErrLockAcquisitionFailed = errors.New("failed to acquire lock")
	ErrInvalidLockIntent     = errors.New("lock intent must not be nil")
	ErrEmptyNonce            = errors.New("lock nonce must not be empty")
	ErrInvalidLease          = errors.New("lock lease must be positive")
	ErrMySQLRequired         = errors.New("dlock requires MySQL configuration")
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
	Key string // The unique identifier for the lock.
	// Nonce must be nonempty and unique to one holding period. Reuse it only for
	// renewals; generate a new value when acquiring after release or lease loss.
	Nonce string
	Lease time.Duration // The length of the lease duration.
}

func NewLockIntent(key, nonce string, lease time.Duration) *LockIntent {
	return &LockIntent{Key: key, Nonce: nonce, Lease: lease}
}

func (li *LockIntent) validate() error {
	if li == nil {
		return ErrInvalidLockIntent
	}
	return validateNonce(li.Nonce)
}

func validateNonce(nonce string) error {
	if nonce == "" {
		return ErrEmptyNonce
	}
	return nil
}

func validateLease(lease time.Duration) error {
	if lease <= 0 {
		return ErrInvalidLease
	}
	return nil
}

// LockManager manages distributed locks using a backend storage system.
type LockManager struct {
	backend Backend
}

// NewLockManager creates a new LockManager with the provided backend.
func NewLockManager(be Backend) *LockManager {
	return &LockManager{backend: be}
}

// NewLockManagerFromViper opens a MySQL manager and exits the process on failure.
// Use OpenLockManagerFromViper to handle initialization errors.
func NewLockManagerFromViper() *LockManager {
	manager, err := OpenLockManagerFromViper()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize distributed lock manager")
	}
	return manager
}

// OpenLockManagerFromViper opens a MySQL manager using the store configuration.
func OpenLockManagerFromViper() (*LockManager, error) {
	var conf store.Config
	if err := viper.UnmarshalKey("store", &conf); err != nil {
		return nil, err
	}
	return NewLockManagerFromConfig(conf)
}

// NewLockManagerFromConfig opens a MySQL manager with the supplied pool settings.
// MySQL takes precedence if both MySQL and SQLite are configured.
func NewLockManagerFromConfig(conf store.Config) (*LockManager, error) {
	if conf.Mysql == nil {
		return nil, ErrMySQLRequired
	}
	db, err := conf.OpenOrCreate(&Dlock{})
	if err != nil {
		return nil, err
	}
	return NewLockManager(NewMySQLBackend(db)), nil
}

// Acquire tries to acquire a distributed lock with the given key and lease duration.
// It returns nil error if the lease was successfully obtained.
func (l *LockManager) Acquire(ctx context.Context, li *LockIntent) error {
	if err := li.validate(); err != nil {
		return err
	}
	if err := validateLease(li.Lease); err != nil {
		return err
	}
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
	if err := li.validate(); err != nil {
		return err
	}
	ok, err := l.backend.DelEntry(ctx, li.Key, li.Nonce)
	if err != nil {
		return errors.WithMessage(err, "failed to delete lock entry")
	}

	if !ok { // lock not held
		return ErrLockNotHeld
	}

	return nil
}
