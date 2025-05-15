package health

import "time"

type TimedCounterConfig struct {
	Threshold time.Duration `default:"1m"` // report unhealthy if threshold reached
	Remind    time.Duration `default:"5m"` // remind unhealthy if unrecovered for a long time
}

// TimedCounter represents an error tolerant health counter, which allows failures in short time
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
// `recovered`: indicates if recovered from unhealthy status.
//
// `elapsed`: indicates the duration since the first failure time.
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
// `unhealthy`: indicates continous failures in a long time.
//
// `unrecovered`: indicates continous failures and unrecovered in a long time.
//
// `elapsed`: indicates the duration since the first failure time.
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
