package dlock

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeBackend struct{ err error }

func (f *fakeBackend) WriteEntry(context.Context, string, string, time.Duration) (bool, error) {
	return false, f.err
}
func (f *fakeBackend) DelEntry(context.Context, string, string) (bool, error) { return false, f.err }
func TestBackendErrors(t *testing.T) {
	sentinel := errors.New("database unavailable")
	m := NewLockManager(&fakeBackend{err: sentinel})
	if !errors.Is(m.Acquire(context.Background(), NewLockIntent("key", "nonce", time.Second)), sentinel) {
		t.Fatal("lost backend error")
	}
}
