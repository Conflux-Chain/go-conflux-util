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
	// Thresholds of max continuous errors for different logging levels. Disabled
	// if the value is 0.
	TraceThreshold int64
	DebugThreshold int64
	InfoThreshold  int64 `default:"1"`
	WarnThreshold  int64 `default:"20"`
	ErrorThreshold int64 `default:"50"`
}

// ErrorTolerantLogger is a thread-safe logger with error tolerance behavior based on
// the continuous error count.
type ErrorTolerantLogger struct {
	conf ErrorToleranceConfig
	// The counter for continuous errors.
	errorCount int64
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
		atomic.StoreInt64(&etl.errorCount, 0)
		return
	}

	errCnt := atomic.AddInt64(&etl.errorCount, 1)
	lvl := etl.determineLevel(errCnt)

	l.WithError(err).Logf(lvl, msg, args...)
}

// determineLevel calculates a log level based on the continuous errors count.
func (etl *ErrorTolerantLogger) determineLevel(errCnt int64) logrus.Level {
	switch {
	case etl.conf.ErrorThreshold > 0 && errCnt >= etl.conf.ErrorThreshold:
		return logrus.ErrorLevel
	case etl.conf.WarnThreshold > 0 && errCnt >= etl.conf.WarnThreshold:
		return logrus.WarnLevel
	case etl.conf.InfoThreshold > 0 && errCnt >= etl.conf.InfoThreshold:
		return logrus.InfoLevel
	case etl.conf.DebugThreshold > 0 && errCnt >= etl.conf.DebugThreshold:
		return logrus.DebugLevel
	case etl.conf.TraceThreshold > 0 && errCnt >= etl.conf.TraceThreshold:
		return logrus.TraceLevel
	default:
		return logrus.ErrorLevel
	}
}
