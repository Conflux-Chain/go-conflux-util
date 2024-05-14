package terminiate

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"
)

// Context enhances the standard context.Context to facilitate a graceful shutdown
// procedure. It includes a sync.WaitGroup to track background tasks that need to
// complete before shutdown and integrates with OS signal handling for SIGTERM and SIGINT.
type Context struct {
	context.Context                 // Embeds standard context for access to its features.
	wg              *sync.WaitGroup // Tracks active goroutines for proper shutdown synchronization.
}

// NewContext constructs a new Context instance wrapping the given parent context
// and using the specified WaitGroup to manage goroutines.
func NewContext(parent context.Context, wg *sync.WaitGroup) *Context {
	return &Context{Context: parent, wg: wg}
}

// Wait sets up a listener for termination signals (SIGTERM and SIGINT) and reacts
// to them by initiating a graceful shutdown sequence. This involves canceling
// the underlying context and waiting for all registered goroutines to finish.
func (c *Context) Wait(cancel context.CancelFunc) {
	// Channel to capture OS termination signals.
	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGTERM, syscall.SIGINT) // Registers signal handlers.

	// Blocks until a termination signal is caught.
	<-termChan

	// Logs that a termination signal was received and begins shutdown.
	logrus.Info("SIGTERM/SIGINT received, initiating shutdown process...")

	// Cancels the context to propagate shutdown signal to children.
	cancel()

	// Awaits the completion of all tracked goroutines.
	c.wg.Wait()

	// Logs upon completion of the graceful shutdown process.
	logrus.Info("Graceful shutdown complete.")
}
