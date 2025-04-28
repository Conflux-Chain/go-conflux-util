package cmd

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"
)

// FatalOnErr adds fatal level log and exit program if the given err is not nil.
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

// GracefulShutdown allows to clean up and shutdown program when termination signal captured.
func GracefulShutdown(wg *sync.WaitGroup, cancel context.CancelFunc) {
	termCh := make(chan os.Signal, 1)
	signal.Notify(termCh, syscall.SIGTERM, syscall.SIGINT)

	<-termCh
	logrus.Info("SIGTERM/SIGINT received, start to shutdown program")

	cancel()
	wg.Wait()
	logrus.Info("Shutdown gracefully")
}
