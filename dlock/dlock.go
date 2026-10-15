// Package dlock implements fail-closed, expiring MySQL leases. A lease does not
// fence business writes: downstream resources must enforce their own fencing.
package dlock

import (
	"context"
	"crypto/rand"
	"errors"
	"reflect"
	"sync"
	"time"
)

var (
	ErrLockNotHeld           = errors.New("lock not held")
	ErrLockAcquisitionFailed = errors.New("failed to acquire lock")
	ErrLeaseExpired          = errors.New("lease expired or outcome uncertain")
	ErrInvalidInput          = errors.New("invalid lock input")
	ErrClosed                = errors.New("lock manager closed")
	ErrUnsupported           = errors.New("unsupported lock operation or database")
	ErrSchema                = errors.New("incompatible dlocks schema")
)

// Grant is a backend capability. Keep Token secret. Each acquisition must use a
// fresh token; custom backends must return a conservative monotonic deadline.
type Grant struct {
	Key        string
	Token      [32]byte
	ValidUntil time.Time
}

// Backend must fail closed on uncertainty, never renew expired generations, and
// compare keys and tokens byte-for-byte. Acquire must not reuse caller identity.
type Backend interface {
	Acquire(context.Context, string, time.Duration) (Grant, error)
	Renew(context.Context, Grant, time.Duration) (Grant, error)
	Release(context.Context, Grant) error
}

type LockManager struct {
	backend  Backend
	mu       sync.RWMutex
	closed   bool
	closeFn  func() error
	closeErr error
}

func NewLockManager(be Backend) (*LockManager, error) {
	if be == nil || (reflect.ValueOf(be).Kind() == reflect.Ptr && reflect.ValueOf(be).IsNil()) {
		return nil, ErrInvalidInput
	}
	return &LockManager{backend: be}, nil
}

func newToken() ([32]byte, error) {
	var token [32]byte
	_, err := rand.Read(token[:])
	return token, err
}

func (m *LockManager) Acquire(ctx context.Context, key string, ttl time.Duration) (*Lease, error) {
	if err := validate(ctx, key, ttl); err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrInvalidInput
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.closed {
		return nil, ErrClosed
	}
	g, err := m.backend.Acquire(ctx, key, ttl)
	if err != nil {
		return nil, err
	}
	if g.Key != key || g.Token == ([32]byte{}) || !time.Now().Before(g.ValidUntil) {
		return nil, ErrLeaseExpired
	}
	return &Lease{manager: m, grant: g}, nil
}

// Lease owns exactly one acquisition generation. Do not copy it.
type Lease struct {
	mu       sync.Mutex
	manager  *LockManager
	grant    Grant
	terminal bool
}

func (l *Lease) ValidUntil() time.Time {
	if l == nil {
		return time.Time{}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.terminal {
		return time.Time{}
	}
	return l.grant.ValidUntil
}
func (l *Lease) Remaining() time.Duration {
	d := time.Until(l.ValidUntil())
	if d < 0 {
		return 0
	}
	return d
}
func (l *Lease) Renew(ctx context.Context, ttl time.Duration) error {
	if l == nil || l.manager == nil {
		return ErrInvalidInput
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := validate(ctx, l.grant.Key, ttl); err != nil {
		return err
	}
	if l.terminal || !time.Now().Before(l.grant.ValidUntil) {
		return ErrLeaseExpired
	}
	m := l.manager
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.closed {
		return ErrClosed
	}
	g, err := m.backend.Renew(ctx, l.grant, ttl)
	if err != nil {
		l.terminal = true
		return err
	}
	if g.Key != l.grant.Key || g.Token != l.grant.Token || !time.Now().Before(g.ValidUntil) {
		l.terminal = true
		return ErrLeaseExpired
	}
	l.grant = g
	return nil
}
func (l *Lease) Release(ctx context.Context) error {
	if l == nil || l.manager == nil || ctx == nil {
		return ErrInvalidInput
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.terminal {
		return ErrLockNotHeld
	}
	l.terminal = true
	m := l.manager
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.closed {
		return ErrClosed
	}
	return m.backend.Release(ctx, l.grant)
}
func validate(ctx context.Context, key string, ttl time.Duration) error {
	if ctx == nil || len(key) == 0 || len(key) > 191 || ttl <= time.Microsecond || ttl%time.Microsecond != 0 {
		return ErrInvalidInput
	}
	return ctx.Err()
}
