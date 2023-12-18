package log

import (
	"sync/atomic"

	"github.com/mcuadros/go-defaults"
	"github.com/sirupsen/logrus"
)

// ErrorToleranceConfig defines the configuration for error tolerance behavior.
type ErrorToleranceConfig struct {
	// Thresholds of max continuous errors for different logging levels. Disabled
	// if the value is 0.
	traceThreshold int64
	debugThreshold int64
	infoThreshold  int64 `default:"1"`
	warnThreshold  int64 `default:"20"`
	errorThreshold int64 `default:"50"`
}

// ErrorTolerantLogger is a thread-safe logger with error tolerance behavior based on
// the continuous error count.
type ErrorTolerantLogger struct {
	*ErrorToleranceConfig
	// The counter for continuous errors.
	errorCount atomic.Int64
}

func NewErrorTolerantLogger(opts ...func(*ErrorTolerantLogger)) *ErrorTolerantLogger {
	conf := &ErrorToleranceConfig{}
	defaults.SetDefaults(conf)

	l := &ErrorTolerantLogger{ErrorToleranceConfig: conf}
	for i := range opts {
		opts[i](l)
	}

	return l
}

// WithTraceThreshold sets the threshold for logging at trace level.
func WithTraceThreshold(threshold int64) func(*ErrorTolerantLogger) {
	return func(l *ErrorTolerantLogger) {
		l.traceThreshold = threshold
	}
}

// WithDebugThreshold sets the threshold for logging at debug level.
func WithDebugThreshold(threshold int64) func(*ErrorTolerantLogger) {
	return func(l *ErrorTolerantLogger) {
		l.debugThreshold = threshold
	}
}

// WithInfoThreshold sets the threshold for logging at info level.
func WithInfoThreshold(threshold int64) func(*ErrorTolerantLogger) {
	return func(l *ErrorTolerantLogger) {
		l.infoThreshold = threshold
	}
}

// WithWarnThreshold sets the threshold for logging at warn level.
func WithWarnThreshold(threshold int64) func(*ErrorTolerantLogger) {
	return func(l *ErrorTolerantLogger) {
		l.warnThreshold = threshold
	}
}

// WithErrorThreshold sets the threshold for logging at error level.
func WithErrorThreshold(threshold int64) func(*ErrorTolerantLogger) {
	return func(l *ErrorTolerantLogger) {
		l.errorThreshold = threshold
	}
}

// Log logs the error message with appropriate level based on the continuous error count.
func (l *ErrorTolerantLogger) Log(err error, logf func(logrus.Level)) {
	// Reset continuous error count if error is nil.
	if err == nil {
		l.errorCount.Store(0)
		return
	}

	errCnt := l.errorCount.Add(1)
	logf(l.determineLevel(errCnt))
}

// determineLevel calculates a log level based on the continuous errors count.
func (l *ErrorTolerantLogger) determineLevel(errCnt int64) logrus.Level {
	switch {
	case l.errorThreshold > 0 && errCnt >= l.errorThreshold:
		return logrus.ErrorLevel
	case l.warnThreshold > 0 && errCnt >= l.warnThreshold:
		return logrus.WarnLevel
	case l.infoThreshold > 0 && errCnt >= l.infoThreshold:
		return logrus.InfoLevel
	case l.debugThreshold > 0 && errCnt >= l.debugThreshold:
		return logrus.DebugLevel
	case l.traceThreshold > 0 && errCnt >= l.traceThreshold:
		return logrus.TraceLevel
	default:
		return logrus.ErrorLevel
	}
}
