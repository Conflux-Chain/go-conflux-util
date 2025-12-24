package health

import (
	"reflect"

	"github.com/mcuadros/go-defaults"
	"github.com/sirupsen/logrus"
)

type CounterConfig struct {
	Threshold uint64 `default:"60"` // report unhealthy if threshold reached
	Remind    uint64 `default:"60"` // remind unhealthy if unrecovered for a long time
}

func DefaultCounterConfig() CounterConfig {
	var config CounterConfig
	defaults.SetDefaults(&config)
	return config
}

// Counter represents an error tolerant health counter, which allows continuous failures in a short time
// and periodically remind unhealthy if unrecovered in time.
type Counter struct {
	config   CounterConfig
	failures uint64
}

func NewCounter(config ...CounterConfig) *Counter {
	var cc CounterConfig

	if len(config) > 0 {
		cc = config[0]
	} else {
		cc = DefaultCounterConfig()
	}

	return &Counter{
		config: cc,
	}
}

// IsSuccess indicates whether any failure occurred.
func (counter *Counter) IsSuccess() bool {
	return counter.failures == 0
}

// OnSuccess erases failure status and return recover information if any.
//
//	@return recovered bool - indicates whether recovered from unhealthy status.
//	@return failures uint64 - indicates the number of continuous failures before success.
func (counter *Counter) OnSuccess() (recovered bool, failures uint64) {
	// last time was success status
	if counter.failures == 0 {
		return
	}

	// report health now after a long time
	if failures = counter.failures; failures >= counter.config.Threshold {
		recovered = true
	}

	// reset
	counter.failures = 0

	return
}

// OnFailure marks failure status and return unhealthy information.
//
//	@return unhealthy bool - indicates whether continuous failures occurred in a long time.
//	@return unrecovered bool - indicates whether continuous failures occurred and unrecovered in a long time.
//	@return failures uint64 - indicates the number of continuous failures so far.
func (counter *Counter) OnFailure() (unhealthy bool, unrecovered bool, failures uint64) {
	counter.failures++

	// error tolerant in short time
	if failures = counter.failures; failures < counter.config.Threshold {
		return
	}

	if delta := failures - counter.config.Threshold; delta == 0 {
		unhealthy = true
	} else if delta%counter.config.Remind == 0 {
		unrecovered = true
	}

	return
}

// OnError updates health status for the given `err` and returns health information.
//
//	@return recovered bool - indicates whether recovered from unhealthy status when `err` is nil.
//	@return unhealthy bool - indicates whether continuous failures occurred in a long time when `err` is not nil.
//	@return unrecovered bool - indicates whether continuous failures occurred and unrecovered in a long time when `err` is not nil.
//	@return failures uint64 - indicates the number of continuous failures so far.
func (counter *Counter) OnError(err error) (recovered bool, unhealthy bool, unrecovered bool, failures uint64) {
	if isErrorNil(err) {
		recovered, failures = counter.OnSuccess()
	} else {
		unhealthy, unrecovered, failures = counter.OnFailure()
	}

	return
}

// LogOnError updates health status for the given `err` and logs health information.
func (counter *Counter) LogOnError(err error, task string) {
	recovered, unhealthy, unrecovered, failures := counter.OnError(err)

	if recovered {
		logrus.WithField("task", task).WithField("failures", failures).Warn("Task is healthy now")
	} else if unhealthy {
		logrus.WithError(err).WithField("task", task).WithField("failures", failures).Warn("Task became unhealthy")
	} else if unrecovered {
		logrus.WithError(err).WithField("task", task).WithField("failures", failures).Warn("Task is not recovered in a long time")
	}
}

func isErrorNil(err error) bool {
	if err == nil {
		return true
	}

	switch reflect.TypeOf(err).Kind() {
	case reflect.Pointer:
		// e.g. err = (*SomeError)(nil)
		return reflect.ValueOf(err).IsNil()
	default:
		return false
	}
}
