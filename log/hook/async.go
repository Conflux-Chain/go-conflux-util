package hook

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

type runningStatus uint32

const (
	statusStopped runningStatus = iota
	statusStarted
)

var (
	errBufferFull = errors.New("async hook buffer is full")
	errNotRunning = errors.New("async hook is not yet running")
)

// Async hook configuration.
type AsyncOption struct {
	// The number of goroutines kept in a pool while idle.
	IdlePoolWorkers uint32 `default:"4"`
	// The max number of goroutines that can be running concurrently while busy.
	MaxConcurrentWorkers uint32 `default:"16"`
	// The maximum number of jobs that can be queued before processing begins.
	JobQueueSize uint32 `default:"128"`
}

// AsyncWrapper is an async wrapper that uses goroutines to invoke the wrapped hook.
type AsyncWrapper struct {
	logrus.Hook
	AsyncOption

	// The flag to indicate running status: 0 - stopped, 1 - running.
	runningFlag uint32
	// The number of currently running worker goroutines.
	numWorkers uint32
	// Job queue for log entries that will be handled by worker goroutines.
	jobQueue chan *logrus.Entry
	// Keeps tracking of worker goroutines.
	workerTracker sync.WaitGroup

	// The context and cancel function for life cycle control.
	ctx    context.Context
	cancel context.CancelFunc
}

func NewAsyncWrapper(wrappedHook logrus.Hook, opts AsyncOption) *AsyncWrapper {
	return &AsyncWrapper{
		AsyncOption: opts,
		Hook:        wrappedHook,
		jobQueue:    make(chan *logrus.Entry, opts.JobQueueSize),
	}
}

func (w *AsyncWrapper) Fire(entry *logrus.Entry) error {
	if !w.isRunning() {
		return errNotRunning
	}

	select {
	case w.jobQueue <- entry:
		return nil
	case <-w.ctx.Done():
		return w.ctx.Err()
	default:
		// job queue is full because workers are too busy or too slow
		// try to boost the workers if possible.
		return w.boost(entry)
	}
}

func (w *AsyncWrapper) Start(ctx context.Context) {
	if w.isRunning() { // already started?
		return
	}

	w.ctx, w.cancel = context.WithCancel(ctx)
	defer w.setRunningFlag(statusStarted)

	for i := uint32(0); i < w.IdlePoolWorkers; i++ {
		w.workerTracker.Add(1)
		w.incrWorkerCounter()

		go func() {
			defer w.workerTracker.Done()
			defer w.decrWorkerCounter()

			for {
				select {
				case entry := <-w.jobQueue:
					w.wrapCall(entry)
				case <-w.ctx.Done():
					return
				}
			}
		}()
	}
}

func (w *AsyncWrapper) Stop() {
	if !w.isRunning() { // not started?
		return
	}

	w.setRunningFlag(statusStopped)

	// cancel and wait for all workers to complete and exit
	w.cancel()
	w.workerTracker.Wait()
}

// boost starts extra goroutine to help empty out the job queue.
func (w *AsyncWrapper) boost(entry *logrus.Entry) error {
	if w.incrWorkerCounter() > w.MaxConcurrentWorkers {
		w.decrWorkerCounter()
		return errBufferFull
	}

	w.workerTracker.Add(1)
	go func() {
		defer w.workerTracker.Done()
		defer w.decrWorkerCounter()

		enq := w.jobQueue
		for {
			select {
			case entry := <-w.jobQueue:
				w.wrapCall(entry)
			case enq <- entry:
				enq, entry = nil, nil
			case <-w.ctx.Done():
				return
			default:
				return
			}
		}
	}()

	return nil
}

func (w *AsyncWrapper) wrapCall(entry *logrus.Entry) {
	if err := w.Hook.Fire(entry); err != nil {
		formatter := logrus.JSONFormatter{}
		entryStr, _ := formatter.Format(entry)

		fmt.Printf("Asyn hook call error %v with entry %v\n", err, entryStr)
	}
}

// set running flag status
func (w *AsyncWrapper) setRunningFlag(status runningStatus) {
	atomic.StoreUint32(&w.runningFlag, uint32(status))
}

// check if the async hook is running or not
func (w *AsyncWrapper) isRunning() bool {
	return atomic.LoadUint32(&w.runningFlag) == uint32(statusStarted)
}

// decrement the number of job workers by 1
func (w *AsyncWrapper) decrWorkerCounter() uint32 {
	return atomic.AddUint32(&w.numWorkers, ^uint32(0))
}

// increment the number of job workers by 1
func (w *AsyncWrapper) incrWorkerCounter() uint32 {
	return atomic.AddUint32(&w.numWorkers, 1)
}
