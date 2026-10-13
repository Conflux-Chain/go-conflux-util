package cmd

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"
)

// FatalIfErr logs err and msg at fatal level if err is not nil; otherwise it returns.
// Only the first optional fields map is used.
// With logrus's default exit function, it exits with code 1 without running defers.
func FatalIfErr(err error, msg string, fields ...logrus.Fields) {
	if err == nil {
		return
	}

	if len(fields) > 0 {
		logrus.WithError(err).WithFields(fields[0]).Fatal(msg)
	} else {
		logrus.WithError(err).Fatal(msg)
	}
}

// GracefulShutdown blocks until SIGTERM or SIGINT, calls cancel, then waits for wg.
// Both wg and cancel must be non-nil. The caller must register tasks with wg.Add
// before calling this function; tasks must respond to cancellation and call wg.Done
// when finished.
//
// The wait has no timeout, and further signals do not trigger a forced exit here.
// The signal subscription is released on return. This function does not exit the
// process; the caller controls what happens after all tasks finish.
func GracefulShutdown(wg *sync.WaitGroup, cancel context.CancelFunc) {
	termCh := make(chan os.Signal, 1)
	signal.Notify(termCh, syscall.SIGTERM, syscall.SIGINT)
	// Unregister on return; the private channel does not need to be closed.
	defer signal.Stop(termCh)

	<-termCh
	logrus.Info("SIGTERM/SIGINT received, start to shutdown program")

	cancel()
	wg.Wait()
	logrus.Info("Shutdown gracefully")
}
