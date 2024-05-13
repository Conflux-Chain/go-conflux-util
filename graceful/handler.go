package graceful

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// ShutdownHandler manages the graceful shutdown process for an application by
// providing a context that can be cancelled to signal termination and a wait group
// to synchronize the completion of ongoing goroutines.
type ShutdownHandler struct {
	ctx       context.Context    // Context that signals shutdown.
	cancelFn  context.CancelFunc // Function to cancel the context and initiate shutdown.
	waitGroup sync.WaitGroup     // Synchronizes goroutines to wait for their completion during shutdown.
}

// NewShutdownHandler initializes a new ShutdownHandler instance configured to
// respond to OS termination signals (SIGTERM, SIGINT) by gracefully shutting down.
func NewShutdownHandler() *ShutdownHandler {
	// Create a cancellable context derived from the background context.
	ctx, cancel := context.WithCancel(context.Background())

	// Channel to receive termination signals.
	termChan := make(chan os.Signal, 1)

	// Register for SIGTERM and SIGINT signals to trigger a graceful shutdown.
	signal.Notify(termChan, syscall.SIGTERM, syscall.SIGINT)

	// Launch a goroutine to listen for termination signals and cancel the context.
	// This ensures the application can react to these signals asynchronously.
	go func() {
		<-termChan // Block until a termination signal is received.

		fmt.Println("SIGTERM/SIGINT received, shutdown process initiated")
		cancel() // Signal shutdown by cancelling the context.
	}()

	return &ShutdownHandler{
		ctx:       ctx,
		cancelFn:  cancel,
		waitGroup: sync.WaitGroup{},
	}
}

// Done returns a channel that's closed when the shutdown is initiated, signaling tasks to stop.
func (h *ShutdownHandler) Done() <-chan struct{} {
	return h.ctx.Done()
}

// Context retrieves the managed context used for signaling shutdown and coordination.
func (h *ShutdownHandler) Context() context.Context {
	return h.ctx
}

// Wait blocks until all registered tasks have finished and then cancels the context.
// This is a crucial step to ensure all resources are released properly.
func (h *ShutdownHandler) Wait() {
	h.waitGroup.Wait() // Block until all tracked goroutines have finished.
	h.cancelFn()       // Cancel the context after all tasks are done to release resources.
}

// Acquire increments the wait group counter, indicating the start of a new set of tasks.
func (h *ShutdownHandler) Acquire(delta int) {
	h.waitGroup.Add(delta) // Increment the counter by delta for each new task.
}

// Release decrements the wait group counter, signifying the completion of a task.
// It's essential to call this when a task that was added with Acquire finishes.
func (h *ShutdownHandler) Release() {
	h.waitGroup.Done() // Decrement the counter, indicating a task's completion.
}
