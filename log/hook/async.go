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
	NumWorkers  int           `default:"1"`  // Number of worker goroutines.
	QueueSize   int           `default:"60"` // Maximum number of queued jobs.
	StopTimeout time.Duration `default:"5s"` // Timeout before forced exit of async processing.
}

// AsyncHook is a logrus hook that processes log entries asynchronously.
type AsyncHook struct {
	logrus.Hook                       // Embedded logrus hook.
	AsyncOption                       // Embedded options.
	mu           sync.Mutex           // Synchronizes access to `healthStatus`.
	started      atomic.Bool          // Atomic flag indicating hook startup state.
	healthStatus health.Counter       // Tracks queue health.
	HealthConfig health.CounterConfig // Configuration for health tracking.

	jobQueue chan *logrus.Entry // Buffered channel for enqueuing log entries.
}

// NewAsyncHookWithCtx initializes and starts a new AsyncHook instance that integrates with
// graceful shutdown handling.
// It's designed to work harmoniously with the application's shutdown process to ensure
// no logs are lost during shutdown.
func NewAsyncHookWithCtx(
	ctx context.Context, wg *sync.WaitGroup, hook logrus.Hook, opts AsyncOption) *AsyncHook {
	h := newAsyncHook(hook, opts)
	h.startWithCtx(ctx, wg)

	return h
}

// NewAsyncHook initializes and starts a standard AsyncHook without graceful shutdown handling.
// Use this when you don't require integration with a graceful shutdown mechanism.
func NewAsyncHook(hook logrus.Hook, opts AsyncOption) *AsyncHook {
	h := newAsyncHook(hook, opts)
	h.start()

	return h
}

// newAsyncHook is a private constructor that sets up the necessary components for an AsyncHook.
// It should not be used directly; instead, use `NewAsyncHook` or `NewAsyncHookWithCtx`.
func newAsyncHook(hook logrus.Hook, opts AsyncOption) *AsyncHook {
	return &AsyncHook{
		AsyncOption:  opts,
		Hook:         hook,
		jobQueue:     make(chan *logrus.Entry, opts.QueueSize),
		HealthConfig: health.CounterConfig{Remind: uint64(opts.QueueSize)},
	}
}

// Fire implements the logrus.Hook interface, which enqueues log entries for async processing or
// handles them synchronously if necessary.
func (h *AsyncHook) Fire(entry *logrus.Entry) error {
	// Synchronously fire the hook for fatal levels or if the hook is not started.
	if entry.Level <= logrus.FatalLevel || !h.started.Load() {
		return h.Hook.Fire(entry)
	}

	select {
	case h.jobQueue <- entry: // Attempt to enqueue the log entry.
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
	defer h.started.Store(true)

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

		// Waits for all workers before attempting to drain the job queue.
		awg.Wait()

		// Drain remaining jobs in the queue to ensure no logs are lost during shutdown.
		h.drainJobQueue()
	}()
}

// start launches the hook's worker goroutines with no lifecycle managment.
func (h *AsyncHook) start() {
	defer h.started.Store(true) // Mark the hook as started.

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
	h.mu.Lock()
	defer h.mu.Unlock()

	recovered, failures := h.healthStatus.OnSuccess(h.HealthConfig)
	if recovered {
		h.notify("Async hook queue congestion recovered", logrus.Fields{"failures": failures})
	}
}

// onFiredFailure handles failed enqueues and monitors queue congestion.
func (h *AsyncHook) onFiredFailure(err error, entry *logrus.Entry) {
	h.mu.Lock()
	defer h.mu.Unlock()

	unhealthy, uncovered, failures := h.healthStatus.OnFailure(h.HealthConfig)
	if unhealthy {
		h.notify("Async hook queue is congested", logrus.Fields{"failures": failures})
	} else if uncovered {
		h.notify("Async hook queue congestion not recovered", logrus.Fields{"failures": failures})
	}

	h.outputStderr(err, entry)
}

// drainJobQueue processes any remaining log entries in the job queue upon shutdown.
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

// notify sends a message through the hook as a warning.
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
	entryStr, _ := formatter.Format(entry)

	fmt.Fprintf(os.Stderr, "Failed to fire async hook with error: %v for logrus entry: %v\n", err, entryStr)
}
