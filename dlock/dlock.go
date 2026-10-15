package dlock

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/store"
	"github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const leasePrecision = time.Millisecond

var (
	ErrLockNotHeld           = errors.New("lock not held")
	ErrLockAcquisitionFailed = errors.New("failed to acquire lock")
	ErrInvalidLockIntent     = errors.New("lock intent must not be nil")
	ErrEmptyNonce            = errors.New("lock nonce must not be empty")
	ErrInvalidLease          = errors.New("lock lease must be positive")
	ErrMySQLRequired         = errors.New("dlock requires MySQL configuration")
	ErrInvalidLockKey        = errors.New("MySQL lock key exceeds 764 bytes")
	ErrIncompatibleSchema    = errors.New("incompatible dlocks schema; migrate before use")
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

// LockIntent identifies a lock holder and its requested lease.
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

// LockManager manages distributed locks using a backend storage system.
type LockManager struct {
	backend    Backend
	ownedStore *store.Store
	closeOnce  sync.Once
	closeErr   error
}

// NewLockManager uses a caller-owned backend. Close does not close its resources.
func NewLockManager(backend Backend) *LockManager {
	return &LockManager{backend: backend}
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
	var config store.Config
	if err := viper.UnmarshalKey("store", &config); err != nil {
		return nil, err
	}
	return NewLockManagerFromConfig(config)
}

// NewLockManagerFromConfig opens a MySQL manager with the supplied pool settings.
// MySQL takes precedence if both MySQL and SQLite are configured.
// The caller must Close the returned manager when it is no longer in use.
func NewLockManagerFromConfig(config store.Config) (*LockManager, error) {
	if config.Mysql == nil {
		return nil, ErrMySQLRequired
	}
	db, err := config.OpenOrCreate()
	if err != nil {
		return nil, err
	}
	backend := NewMySQLBackend(db)
	manager := NewLockManager(backend)
	manager.ownedStore = store.NewStore(db)
	if err := backend.initializeSchema(context.Background()); err != nil {
		manager.Close()
		return nil, err
	}
	return manager, nil
}

// Close releases only the database pool opened by a configuration constructor.
// Repeated calls return the first close result. Stop using the manager first.
func (manager *LockManager) Close() error {
	manager.closeOnce.Do(func() {
		if manager.ownedStore != nil {
			manager.closeErr = manager.ownedStore.Close()
		}
	})
	return manager.closeErr
}

// Acquire tries to acquire a distributed lock with the given key and lease duration.
// It returns nil only while a conservative lease budget remains. Use
// AcquireWithDeadline to bound protected work by that budget.
func (manager *LockManager) Acquire(ctx context.Context, intent *LockIntent) error {
	_, err := manager.AcquireWithDeadline(ctx, intent)
	return err
}

// AcquireWithDeadline acquires or renews a lease and returns a local monotonic
// deadline. The budget starts before storage access and reserves one millisecond
// for timestamp precision. It assumes the database clock does not jump forward.
// Stop work by this deadline, or earlier if renewal fails. No deadline is returned
// on failure; a successful write with an exhausted budget is conditionally released.
func (manager *LockManager) AcquireWithDeadline(ctx context.Context, intent *LockIntent) (time.Time, error) {
	start := time.Now()
	if err := intent.validate(); err != nil {
		return time.Time{}, err
	}
	if err := validateLease(intent.Lease); err != nil {
		return time.Time{}, err
	}
	deadline := start.Add(intent.Lease - leasePrecision)
	if time.Since(start) >= intent.Lease-leasePrecision {
		return time.Time{}, ErrLockAcquisitionFailed
	}
	ok, err := manager.backend.WriteEntry(ctx, intent.Key, intent.Nonce, intent.Lease)
	if err != nil {
		return time.Time{}, errors.WithMessage(err, "failed to write lock entry")
	}

	if !ok {
		return time.Time{}, ErrLockAcquisitionFailed
	}

	if err := finishAcquisition(ctx, manager.backend, intent.Key, intent.Nonce, start, intent.Lease); err != nil {
		return time.Time{}, err
	}
	return deadline, nil
}

// Release attempts to release an existing lock.
// It returns nil error if the lock was successfully released.
func (manager *LockManager) Release(ctx context.Context, intent *LockIntent) error {
	if err := intent.validate(); err != nil {
		return err
	}
	ok, err := manager.backend.DelEntry(ctx, intent.Key, intent.Nonce)
	if err != nil {
		return errors.WithMessage(err, "failed to delete lock entry")
	}

	if !ok {
		return ErrLockNotHeld
	}

	return nil
}

func (intent *LockIntent) validate() error {
	if intent == nil {
		return ErrInvalidLockIntent
	}
	return validateNonce(intent.Nonce)
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

// finishAcquisition confirms a successful write still has a usable lease, or
// conditionally releases it. Cleanup uses a fresh bounded context because
// acquisition may have been canceled.
func finishAcquisition(ctx context.Context, backend Backend, key, nonce string, start time.Time, lease time.Duration) error {
	err := ctx.Err()
	if err == nil && time.Since(start) < lease-leasePrecision {
		return nil
	}
	if err == nil {
		err = ErrLockAcquisitionFailed
	}
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
	defer cancel()
	if _, cleanupErr := backend.DelEntry(cleanupCtx, key, nonce); cleanupErr != nil {
		return fmt.Errorf("%w; failed to discard lease: %w", err, cleanupErr)
	}
	return err
}
