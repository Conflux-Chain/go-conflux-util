package log

import (
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

// TolerantLogger is a logger that tolerates a certain number of error tries before
// escalating to an error level.
type TolerantLogger struct {
	errorLimit uint64
	errorCount atomic.Uint64
}

func NewTolerantLogger(errorLimit int) *TolerantLogger {
	if errorLimit == 0 {
		errorLimit = 1
	}

	return &TolerantLogger{errorLimit: uint64(errorLimit)}
}

func (t *TolerantLogger) Error(logger *logrus.Logger, msg string) {
	t.Errorf(logger, msg)
}

func (t *TolerantLogger) Errorf(logger *logrus.Logger, msg string, args ...interface{}) {
	errCnt := t.errorCount.Add(1)
	if errCnt%t.errorLimit != 0 {
		logger.Infof(msg, args...)
	} else {
		logger.WithField("errCount", errCnt).Errorf(msg, args...)
	}
}

func (t *TolerantLogger) ResetErrorCount() {
	t.errorCount.Store(0)
}
