package hook

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

// Async hook configuration.
type AsyncOption struct {
	// Enable async mode or not
	Enabled bool
	// The max number of worker goroutines that can be running concurrently.
	MaxWorkers uint32 `default:"4"`
	// The maximum number of jobs that can be queued before processing begins.
	QueueSize uint32 `default:"32"`
}

// AsyncHook is an async wrapper that uses goroutines to invoke an wrapped hook.
type AsyncHook struct {
	logrus.Hook
	AsyncOption

	// The number of currently running worker goroutines.
	numWorkers uint32
	// Job queue for log entries that will be handled by worker goroutines.
	jobQueue chan *logrus.Entry
	// The context for life cycle control.
	ctx context.Context
}

func NewAsyncHook(ctx context.Context, hook logrus.Hook, opts AsyncOption) *AsyncHook {
	h := &AsyncHook{
		AsyncOption: opts,
		Hook:        hook,
		ctx:         ctx,
		jobQueue:    make(chan *logrus.Entry, opts.QueueSize),
	}

	h.start()
	return h
}

func (w *AsyncHook) Fire(entry *logrus.Entry) error {
	if entry.Level <= logrus.FatalLevel {
		// For `fatal` or `panic` level logging, we have to fire the wrapped hook
		// synchronously, otherwise the hook would never be triggered due to
		// application life cycle.
		return w.Hook.Fire(entry)
	}

	select {
	case w.jobQueue <- entry:
		return nil
	case <-w.ctx.Done():
		return nil
	default:
		// job queue is full because workers are too busy or too slow
		// try to boost the workers if possible.
		return w.boost(entry)
	}
}

func (h *AsyncHook) start() {
	h.incrWorkerCounter()

	go func() {
		defer h.decrWorkerCounter()
		for {
			select {
			case entry := <-h.jobQueue:
				h.fire(entry)
			case <-h.ctx.Done():
				return
			}
		}
	}()
}

// boost starts extra goroutine to help empty out the job queue.
func (h *AsyncHook) boost(entry *logrus.Entry) error {
	if h.incrWorkerCounter() > h.MaxWorkers {
		h.decrWorkerCounter()
		return errors.New("async hook buffer is full")
	}

	go func() {
		defer h.decrWorkerCounter()

		enq := h.jobQueue
		for {
			select {
			case entry := <-h.jobQueue:
				h.fire(entry)
			case enq <- entry:
				enq, entry = nil, nil
			case <-h.ctx.Done():
				return
			default:
				return
			}
		}
	}()

	return nil
}

func (h *AsyncHook) fire(entry *logrus.Entry) {
	if err := h.Hook.Fire(entry); err != nil {
		formatter := logrus.TextFormatter{}
		entryStr, _ := formatter.Format(entry)

		fmt.Fprintf(os.Stderr, "Asyn hook fire error %v with entry %v\n", err, entryStr)
	}
}

// decrement the number of job workers by 1
func (h *AsyncHook) decrWorkerCounter() uint32 {
	return atomic.AddUint32(&h.numWorkers, ^uint32(0))
}

// increment the number of job workers by 1
func (h *AsyncHook) incrWorkerCounter() uint32 {
	return atomic.AddUint32(&h.numWorkers, 1)
}
