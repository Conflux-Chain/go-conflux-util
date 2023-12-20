package log

import (
	"sync/atomic"

	"github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/mcuadros/go-defaults"
	"github.com/sirupsen/logrus"
)

var (
	// Default error tolerance config
	DefaultETConfig ErrorToleranceConfig
	// Default error tolerance logger
	DefaultETLogger *ErrorTolerantLogger
)

func init() {
	defaults.SetDefaults(&DefaultETConfig)
	DefaultETLogger = NewErrorTolerantLogger(DefaultETConfig)
}

// ErrorToleranceConfig defines the configuration for error tolerance behavior.
type ErrorToleranceConfig struct {
	ReportFailures uint64 `default:"60"`
	RemindFailures uint64 `default:"60"`
}

// ErrorTolerantLogger is a thread-safe logger with error tolerance behavior based on
// the continuous error count.
type ErrorTolerantLogger struct {
	conf ErrorToleranceConfig
	// The counter for continuous errors.
	errorCount uint64
}

func NewErrorTolerantLogger(conf ErrorToleranceConfig) *ErrorTolerantLogger {
	return &ErrorTolerantLogger{conf: conf}
}

func MustNewErrorTolerantLoggerFromViper() *ErrorTolerantLogger {
	var config ErrorToleranceConfig
	viper.MustUnmarshalKey("log.errorTolerance", &config)
	return NewErrorTolerantLogger(config)
}

// Log logs the error message with appropriate level based on the continuous error count.
func (etl *ErrorTolerantLogger) Log(l logrus.FieldLogger, err error, msg string) {
	etl.Logf(l, err, msg)
}

func (etl *ErrorTolerantLogger) Logf(l logrus.FieldLogger, err error, msg string, args ...interface{}) {
	// Reset continuous error count if error is nil.
	if err == nil {
		atomic.StoreUint64(&etl.errorCount, 0)
		return
	}

	// do not report error for temp failures
	errCnt := atomic.AddUint64(&etl.errorCount, 1)
	if errCnt < etl.conf.ReportFailures {
		return
	}

	// report error for continuous failures
	if (errCnt-etl.conf.ReportFailures)%etl.conf.RemindFailures == 0 {
		l.WithError(err).WithField("errCounter", errCnt).Errorf(msg, args...)
	}
}
