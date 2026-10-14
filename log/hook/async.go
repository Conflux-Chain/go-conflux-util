package hook

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/health"
	"github.com/sirupsen/logrus"
)

// ErrAsyncQueueFull is the error returned when the async queue is full.
var ErrAsyncQueueFull = errors.New("async hook queue is full")

// AsyncOption defines configuration options for the AsyncHook.
type AsyncOption struct {
	NumWorkers int // Number of worker goroutines. Non-positive values disable async mode (default: 0).
	QueueSize  int `default:"60"` // Maximum number of queued jobs.

	// StopTimeout is a best-effort queue draining budget after all workers exit.
	// It excludes waiting for workers and cannot interrupt hook calls or notifications,
	// so it does not bound the total shutdown time.
	StopTimeout time.Duration `default:"5s"`
}

// AsyncHook is a logrus hook that processes log entries asynchronously.
type AsyncHook struct {
	logrus.Hook                  // Embedded logrus hook.
	AsyncOption                  // Embedded options.
	started      atomic.Bool     // Atomic flag indicating hook startup state.
	healthStatus *health.Counter // Tracks queue health with its own internal synchronization.

	jobQueue chan *logrus.Entry // Buffered channel for enqueuing log entries.
}

// NewAsyncHookWithCtx initializes and starts a new AsyncHook instance that integrates with
// graceful shutdown handling.
// On cancellation, it waits for workers to exit, then attempts to drain queued entries.
// StopTimeout does not bound wg.Wait; the underlying hook must limit its own I/O duration.
// Stop and wait for log producers before cancelling ctx to avoid enqueuing during shutdown.
// Delivery is best effort: queue overflow, drain timeout, or send failures can lose alerts.
// If opts.NumWorkers <= 0, entries are handled synchronously without starting background work.
func NewAsyncHookWithCtx(
	ctx context.Context, wg *sync.WaitGroup, hook logrus.Hook, opts AsyncOption) *AsyncHook {
	h := newAsyncHook(hook, opts)
	if opts.NumWorkers > 0 {
		h.startWithCtx(ctx, wg)
	}

	return h
}

// NewAsyncHook initializes and starts a standard AsyncHook without graceful shutdown handling.
// Use this when you don't require integration with a graceful shutdown mechanism.
// If opts.NumWorkers <= 0, entries are handled synchronously without starting background work.
func NewAsyncHook(hook logrus.Hook, opts AsyncOption) *AsyncHook {
	h := newAsyncHook(hook, opts)
	if opts.NumWorkers > 0 {
		h.start()
	}

	return h
}

// newAsyncHook is a private constructor that sets up the necessary components for an AsyncHook.
// It should not be used directly; instead, use `NewAsyncHook` or `NewAsyncHookWithCtx`.
func newAsyncHook(hook logrus.Hook, opts AsyncOption) *AsyncHook {
	return &AsyncHook{
		AsyncOption: opts,
		Hook:        hook,
		jobQueue:    make(chan *logrus.Entry, opts.QueueSize),
		healthStatus: health.NewCounter(health.CounterConfig{
			Threshold: 1,
			Remind:    uint64(opts.QueueSize),
		}),
	}
}

// Fire implements the logrus.Hook interface, which enqueues log entries for async processing or
// handles them synchronously if necessary.
// Queue congestion and recovery notifications are sent synchronously and may delay this call.
// Queued entries have a shallow copy of Data; callers must not concurrently mutate objects
// referenced by its values while the entry is being processed.
func (h *AsyncHook) Fire(entry *logrus.Entry) error {
	// Synchronously fire the hook for fatal levels or if the hook is not started.
	if entry.Level <= logrus.FatalLevel || !h.started.Load() {
		return h.Hook.Fire(entry)
	}

	// Logrus continues using entry.Buffer after Fire returns. Dup copies Data without
	// sharing that buffer, but omits Level, Message and Caller, so restore them here.
	queued := entry.Dup()
	queued.Level = entry.Level
	queued.Message = entry.Message
	queued.Caller = entry.Caller

	select {
	case h.jobQueue <- queued: // Attempt to enqueue the independent log entry.
		h.onFiredSuccess()
		return nil
	default: // if the queue is full, return an error.
		h.onFiredFailure(ErrAsyncQueueFull, entry)
		return ErrAsyncQueueFull
	}
}

// startWithCtx initiates the hook's workers and sets up a mechanism to gracefully
// drain the job queue upon receiving a shutdown signal through the provided context.
func (h *AsyncHook) startWithCtx(ctx context.Context, wg *sync.WaitGroup) {
	// Mark started before launching workers so an early exit cannot be overwritten.
	h.started.Store(true)

	wg.Add(1)
	go func() {
		defer wg.Done()

		var awg sync.WaitGroup
		for i := 0; i < h.NumWorkers; i++ {
			awg.Add(1)
			go func() {
				defer awg.Done()
				h.worker(ctx)
			}()
		}

		// Wait for all workers; StopTimeout has not started and cannot interrupt their hook calls.
		awg.Wait()

		// Attempt to drain remaining jobs within the best-effort StopTimeout budget.
		h.drainJobQueue()
	}()
}

// start launches the hook's worker goroutines without lifecycle managment.
func (h *AsyncHook) start() {
	h.started.Store(true) // Mark the hook as started before launching workers.

	for i := 0; i < h.NumWorkers; i++ {
		go h.worker(context.Background())
	}
}

// worker is the main loop for each worker goroutine, processing log entries from the job queue.
func (h *AsyncHook) worker(ctx context.Context) {
	for {
		select {
		case entry := <-h.jobQueue:
			h.fire(entry)
		case <-ctx.Done():
			h.started.Store(false) // Mark the hook as stopped.
			return
		}
	}
}

// fire triggers the underlying logrus hook and handles any potential errors by outputting to stderr.
func (h *AsyncHook) fire(entry *logrus.Entry) {
	if err := h.Hook.Fire(entry); err != nil {
		h.outputStderr(err, entry)
	}
}

// onFiredSuccess signals successful enqueue and monitors queue health recovery.
func (h *AsyncHook) onFiredSuccess() {
	recovered, failures := h.healthStatus.OnSuccess()
	if recovered {
		h.notify("Async hook queue congestion recovered", logrus.Fields{"failures": failures})
	}
}

// onFiredFailure handles failed enqueues and monitors queue congestion.
func (h *AsyncHook) onFiredFailure(err error, entry *logrus.Entry) {
	unhealthy, uncovered, failures := h.healthStatus.OnFailure()
	if unhealthy {
		h.notify("Async hook queue is congested", logrus.Fields{"failures": failures})
	} else if uncovered {
		h.notify("Async hook queue congestion not recovered", logrus.Fields{"failures": failures})
	}

	h.outputStderr(err, entry)
}

// drainJobQueue attempts to process remaining entries after all workers exit.
// Expiry is checked between hook calls and cannot interrupt them or the final notification.
// When both an entry and expiry are ready, select may still choose the entry.
func (h *AsyncHook) drainJobQueue() {
	ctx, cancel := context.WithTimeout(context.Background(), h.StopTimeout)
	defer cancel()

	for {
		select {
		case entry := <-h.jobQueue:
			h.fire(entry)
		case <-ctx.Done():
			if len(h.jobQueue) > 0 {
				h.notify("Async hook exiting with jobs remaining", logrus.Fields{"numJobs": len(h.jobQueue)})
			}
			return
		default:
			return
		}
	}
}

// notify sends a warning synchronously without holding a queue health lock.
// Concurrent notifications may complete out of order.
func (h *AsyncHook) notify(msg string, fields logrus.Fields) {
	h.fire(&logrus.Entry{
		Time:    time.Now(),
		Level:   logrus.WarnLevel,
		Message: msg,
		Data:    fields,
	})
}

// outputStderr writes the error and the log entry to the standard error output.
func (h *AsyncHook) outputStderr(err error, entry *logrus.Entry) {
	formatter := logrus.TextFormatter{}
	formatted, _ := formatter.Format(entry)

	fmt.Fprintf(os.Stderr, "Failed to fire async hook with error: %v for logrus entry: %s", err, string(formatted))
}
