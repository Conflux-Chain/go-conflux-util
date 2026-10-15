package dlock

import (
	"context"
	"errors"
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
