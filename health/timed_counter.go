package health

import "time"

type TimedCounterConfig struct {
	Threshold time.Duration `default:"1m"` // report unhealthy if threshold reached
	Remind    time.Duration `default:"5m"` // remind unhealthy if unrecovered for a long time
}

// TimedCounter represents an error tolerant health counter, which allows failures in short time
// and periodically remind unhealthy if unrecovered in time.
type TimedCounter struct {
	TimedCounterConfig

	failedAt time.Time // first failure time
	reports  int       // number of times to report unhealthy
}

func NewTimedCounter(config TimedCounterConfig) *TimedCounter {
	return &TimedCounter{
		TimedCounterConfig: config,
	}
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
func (counter *TimedCounter) OnSuccess() (recovered bool, elapsed time.Duration) {
	return counter.onSuccessAt(time.Now())
}

func (counter *TimedCounter) onSuccessAt(now time.Time) (recovered bool, elapsed time.Duration) {
	// last time was success status
	if counter.failedAt.IsZero() {
		return
	}

	// report health now after a long time
	if elapsed = now.Sub(counter.failedAt); elapsed > counter.Threshold {
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
func (counter *TimedCounter) OnFailure() (unhealthy bool, unrecovered bool, elapsed time.Duration) {
	return counter.onFailureAt(time.Now())
}

func (counter *TimedCounter) onFailureAt(now time.Time) (unhealthy bool, unrecovered bool, elapsed time.Duration) {
	// record the first failure time
	if counter.failedAt.IsZero() {
		counter.failedAt = now
	}

	// error tolerant in short time
	if elapsed = now.Sub(counter.failedAt); elapsed <= counter.Threshold {
		return
	}

	// become unhealthy
	if counter.reports == 0 {
		unhealthy = true
		counter.reports++
		return
	}

	// remind time not reached
	if remind := counter.Threshold + counter.Remind*time.Duration(counter.reports); elapsed <= remind {
		return
	}

	// remind unhealthy
	unrecovered = true
	counter.reports++

	return
}
