package hook

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
)

type testHookFunc func(*logrus.Entry) error

func (f testHookFunc) Fire(entry *logrus.Entry) error { return f(entry) }
func (f testHookFunc) Levels() []logrus.Level         { return logrus.AllLevels }

func TestAsyncHookWithoutWorkers(t *testing.T) {
	constructors := []struct {
		name string
		new  func(logrus.Hook, AsyncOption) *AsyncHook
	}{
		{"standard", NewAsyncHook},
		{"with_context", func(hook logrus.Hook, opts AsyncOption) *AsyncHook {
			var wg sync.WaitGroup
			h := NewAsyncHookWithCtx(context.Background(), &wg, hook, opts)
			// With no workers, construction must not leave background work running.
			wg.Wait()
			return h
		}},
	}

	for _, constructor := range constructors {
		for _, workers := range []int{0, -1} {
			t.Run(fmt.Sprintf("%s/workers=%d", constructor.name, workers), func(t *testing.T) {
				entry := &logrus.Entry{Level: logrus.InfoLevel, Message: "test log"}
				calls := 0
				var fireErr error
				underlying := testHookFunc(func(got *logrus.Entry) error {
					calls++
					if got != entry {
						t.Error("underlying hook received a different entry")
					}
					return fireErr
				})
				h := constructor.new(underlying, AsyncOption{NumWorkers: workers, QueueSize: 2})

				for i, wantErr := range []error{nil, errors.New("hook failed")} {
					fireErr = wantErr
					if err := h.Fire(entry); err != wantErr {
						t.Errorf("Fire() error = %v, want %v", err, wantErr)
					}
					if calls != i+1 {
						t.Errorf("underlying hook calls = %d, want %d before Fire returns", calls, i+1)
					}
				}
			})
		}
	}
}
