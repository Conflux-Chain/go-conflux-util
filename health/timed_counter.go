package health

import "time"

type TimedCounterConfig struct {
	Threshold time.Duration `default:"1m"` // report unhealthy if threshold reached
	Remind    time.Duration `default:"5m"` // remind unhealthy if unrecovered for a long time
}

// TimedCounter represents an error tolerant health counter, which allows continuous failures in a short time
// and periodically remind unhealthy if unrecovered in time.
type TimedCounter struct {
	failedAt time.Time // first failure time
	reports  int       // number of times to report unhealthy
}

// IsSuccess indicates whether any failure occurred.
func (counter *TimedCounter) IsSuccess() bool {
	return counter.failedAt.IsZero()
}

// OnSuccess erases failure status and return recover information if any.
//
//   @return recovered bool - indicates whether recovered from unhealthy status.
//   @return elapsed time.Duration - indicates the duration since the first failure time.
func (counter *TimedCounter) OnSuccess(config TimedCounterConfig) (recovered bool, elapsed time.Duration) {
	return counter.onSuccessAt(config, time.Now())
}

func (counter *TimedCounter) onSuccessAt(config TimedCounterConfig, now time.Time) (recovered bool, elapsed time.Duration) {
	// last time was success status
	if counter.failedAt.IsZero() {
		return
	}

	// report health now after a long time
	if elapsed = now.Sub(counter.failedAt); elapsed >= config.Threshold {
		recovered = true
	}

	// reset
	counter.failedAt = time.Time{}
	counter.reports = 0

	return
}

// OnFailure marks failure status and return unhealthy information.
//
//   @return unhealthy bool - indicates whether continuous failures occurred in a long time.
//   @return unrecovered bool - indicates whether continuous failures occurred and unrecovered in a long time.
//   @return elapsed time.Duration - indicates the duration since the first failure time.
func (counter *TimedCounter) OnFailure(config TimedCounterConfig) (unhealthy bool, unrecovered bool, elapsed time.Duration) {
	return counter.onFailureAt(config, time.Now())
}

func (counter *TimedCounter) onFailureAt(config TimedCounterConfig, now time.Time) (unhealthy bool, unrecovered bool, elapsed time.Duration) {
	// record the first failure time
	if counter.failedAt.IsZero() {
		counter.failedAt = now
	}

	// error tolerant in short time
	if elapsed = now.Sub(counter.failedAt); elapsed < config.Threshold {
		return
	}

	// become unhealthy
	if counter.reports == 0 {
		unhealthy = true
		counter.reports++
		return
	}

	// remind time not reached
	if remind := config.Threshold + config.Remind*time.Duration(counter.reports); elapsed < remind {
		return
	}

	// remind unhealthy
	unrecovered = true
	counter.reports++

	return
}

// OnError updates health status for the given `err` and returns health information.
//
//   @return recovered bool - indicates whether recovered from unhealthy status when `err` is nil.
//   @return unhealthy bool - indicates whether continuous failures occurred in a long time when `err` is not nil.
//   @return unrecovered bool - indicates whether continuous failures occurred and unrecovered in a long time when `err` is not nil.
//   @return elapsed time.Duration - indicates the duration since the first failure time.
func (counter *TimedCounter) OnError(config TimedCounterConfig, err error) (recovered bool, unhealthy bool, unrecovered bool, elapsed time.Duration) {
	if isErrorNil(err) {
		recovered, elapsed = counter.OnSuccess(config)
	} else {
		unhealthy, unrecovered, elapsed = counter.OnFailure(config)
	}

	return
}
