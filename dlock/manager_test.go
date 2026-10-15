package dlock

import (
	"context"
	"errors"
	"github.com/Conflux-Chain/go-conflux-util/store"
	"github.com/spf13/viper"
	"sync"
	"testing"
	"time"
)

type fakeBackend struct {
	err   error
	grant Grant
	calls int
}

func (f *fakeBackend) Acquire(_ context.Context, key string, ttl time.Duration) (Grant, error) {
	f.calls++
	if f.err != nil {
		return Grant{}, f.err
	}
	f.grant = Grant{Key: key, Token: [32]byte{1}, ValidUntil: time.Now().Add(ttl)}
	return f.grant, nil
}
func (f *fakeBackend) Renew(_ context.Context, g Grant, ttl time.Duration) (Grant, error) {
	f.calls++
	g.ValidUntil = time.Now().Add(ttl)
	return g, f.err
}
func (f *fakeBackend) Release(context.Context, Grant) error { f.calls++; return f.err }

func TestBackendErrors(t *testing.T) {
	sentinel := errors.New("database unavailable")
	m, _ := NewLockManager(&fakeBackend{err: sentinel})
	if _, err := m.Acquire(context.Background(), "key", time.Second); !errors.Is(err, sentinel) {
		t.Fatal("lost backend error", err)
	}
}
func TestLeaseGeneration(t *testing.T) {
	f := &fakeBackend{}
	m, err := NewLockManager(f)
	if err != nil {
		t.Fatal(err)
	}
	l, err := m.Acquire(context.Background(), "key", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err = l.Renew(context.Background(), time.Second); err != nil {
		t.Fatal(err)
	}
	if err = l.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
	before := f.calls
	if !errors.Is(l.Renew(context.Background(), time.Second), ErrLeaseExpired) {
		t.Fatal("released lease renewed")
	}
	if !errors.Is(l.Release(context.Background()), ErrLockNotHeld) {
		t.Fatal("release was not terminal")
	}
	if f.calls != before {
		t.Fatal("terminal lease reached backend")
	}
}

type delayedBackend struct {
	fakeBackend
	delay time.Duration
}

func (f *delayedBackend) Acquire(ctx context.Context, key string, ttl time.Duration) (Grant, error) {
	time.Sleep(f.delay)
	return f.fakeBackend.Acquire(ctx, key, ttl)
}
func (f *delayedBackend) Renew(ctx context.Context, g Grant, ttl time.Duration) (Grant, error) {
	time.Sleep(f.delay)
	return f.fakeBackend.Renew(ctx, g, ttl)
}
func TestManagerBoundsCustomBackend(t *testing.T) {
	f := &delayedBackend{delay: 30 * time.Millisecond}
	m, _ := NewLockManager(f)
	if l, err := m.Acquire(context.Background(), "key", 10*time.Millisecond); l != nil || !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("usable expired grant: %v %v", l, err)
	}
	f.delay = 0
	l, err := m.Acquire(context.Background(), "key", 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	f.delay = 30 * time.Millisecond
	if err = l.Renew(context.Background(), time.Second); !errors.Is(err, ErrLeaseExpired) {
		t.Fatal("renew finished after previous deadline", err)
	}
	if l.Remaining() != 0 {
		t.Fatal("uncertain lease remained usable")
	}
}

func TestValidationBeforeBackend(t *testing.T) {
	f := &fakeBackend{}
	m, _ := NewLockManager(f)
	cases := []struct {
		key string
		ttl time.Duration
	}{
		{"", time.Second}, {string(make([]byte, 192)), time.Second}, {"key", 0}, {"key", -time.Second}, {"key", time.Nanosecond}, {"key", time.Microsecond}, {"key", time.Second + time.Nanosecond},
	}
	for _, c := range cases {
		if _, err := m.Acquire(context.Background(), c.key, c.ttl); !errors.Is(err, ErrInvalidInput) {
			t.Fatal(c, err)
		}
	}
	if f.calls != 0 {
		t.Fatal("invalid input reached backend")
	}
	if _, err := m.Acquire(nil, "key", time.Second); !errors.Is(err, ErrInvalidInput) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := m.Acquire(ctx, "key", time.Second); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	var nilBackend *fakeBackend
	if _, err := NewLockManager(nilBackend); err != ErrInvalidInput {
		t.Fatal(err)
	}
}

func TestCloseOwnershipAndConcurrency(t *testing.T) {
	m, _ := NewLockManager(&fakeBackend{})
	closes := 0
	m.closeFn = func() error { closes++; return nil }
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := m.Close(); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if closes != 1 {
		t.Fatal("pool closed more than once")
	}
	if _, err := m.Acquire(context.Background(), "key", time.Second); err != ErrClosed {
		t.Fatal(err)
	}
	if _, err := NewLockManagerFromConfig(context.Background(), store.Config{}); err != ErrUnsupported {
		t.Fatal(err)
	}
}
func TestViperFactoryRejectsFallbackAndDecodeError(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	if _, err := NewLockManagerFromViper(); err != ErrUnsupported {
		t.Fatal(err)
	}
	viper.Set("store.sqlite.path", ":memory:")
	if _, err := NewLockManagerFromViper(); err != ErrUnsupported {
		t.Fatal(err)
	}
	viper.Set("store.maxopenconns", "invalid")
	if _, err := NewLockManagerFromViper(); err == nil {
		t.Fatal("decode error ignored")
	}
}
