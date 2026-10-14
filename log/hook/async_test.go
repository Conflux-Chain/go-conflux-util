package hook

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

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

func TestAsyncHookCopiesQueuedEntry(t *testing.T) {
	// Inspect the queued entry without starting workers so no scheduling is needed.
	h := newAsyncHook(testHookFunc(func(*logrus.Entry) error {
		t.Fatal("entry should be queued, not handled synchronously")
		return nil
	}), AsyncOption{QueueSize: 1})
	h.started.Store(true)

	entry := &logrus.Entry{
		Logger:  logrus.New(),
		Data:    logrus.Fields{"request_id": "original"},
		Time:    time.Now(),
		Level:   logrus.WarnLevel,
		Caller:  &runtime.Frame{Function: "example.Handle"},
		Message: "queued-message",
		Buffer:  bytes.NewBufferString("logger-only buffer\n"),
		Context: context.Background(),
	}
	if err := h.Fire(entry); err != nil {
		t.Fatal(err)
	}
	queued := <-h.jobQueue
	if queued == entry {
		t.Error("queued entry shares the original Entry")
	}
	if queued.Buffer != nil {
		t.Error("queued entry retained the logger's buffer")
	}
	if queued.Logger != entry.Logger || queued.Time != entry.Time ||
		queued.Level != entry.Level || queued.Caller != entry.Caller || queued.Context != entry.Context {
		t.Error("queued entry lost log metadata")
	}

	// Later changes to the original entry must not alter the queued log.
	entry.Message = "changed-message"
	entry.Data["request_id"] = "changed"
	if queued.Message != "queued-message" || queued.Data["request_id"] != "original" {
		t.Error("queued log changed when the original entry was modified")
	}
	formatter := logrus.TextFormatter{DisableTimestamp: true, DisableColors: true}
	output, err := formatter.Format(queued)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(output), "logger-only") || strings.Count(string(output), "queued-message") != 1 {
		t.Errorf("unexpected queued log output: %q", output)
	}
}
