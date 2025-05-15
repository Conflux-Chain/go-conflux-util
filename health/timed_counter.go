package health

import "time"

type TimedCounterConfig struct {
	Threshold time.Duration `default:"1m"` // report unhealthy if threshold reached
	Remind    time.Duration `default:"5m"` // remind unhealthy if unrecovered for a long time
}

// TimedCounter represents an error tolerant health counter, which allows continuous failures in a short time
// and periodically remind unhealthy if unrecovered in time.
type TimedCounter struct {
	config   TimedCounterConfig
	failedAt time.Time // first failure time
	reports  int       // number of times to report unhealthy
}

func NewTimedCounter(config TimedCounterConfig) *TimedCounter {
	return &TimedCounter{
		config: config,
	}
}

// IsSuccess indicates whether any failure occurred.
func (counter *TimedCounter) IsSuccess() bool {
	return counter.failedAt.IsZero()
}

// OnSuccess erases failure status and return recover information if any.
//
//   @return recovered bool - indicates whether recovered from unhealthy status.
//   @return elapsed time.Duration - indicates the duration since the first failure time.
func (counter *TimedCounter) OnSuccess() (recovered bool, elapsed time.Duration) {
	return counter.onSuccessAt(time.Now())
}

func (counter *TimedCounter) onSuccessAt(now time.Time) (recovered bool, elapsed time.Duration) {
	// last time was success status
	if counter.failedAt.IsZero() {
		return
	}

	// report health now after a long time
	if elapsed = now.Sub(counter.failedAt); elapsed >= counter.config.Threshold {
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
func (counter *TimedCounter) OnFailure() (unhealthy bool, unrecovered bool, elapsed time.Duration) {
	return counter.onFailureAt(time.Now())
}

func (counter *TimedCounter) onFailureAt(now time.Time) (unhealthy bool, unrecovered bool, elapsed time.Duration) {
	// record the first failure time
	if counter.failedAt.IsZero() {
		counter.failedAt = now
	}

	// error tolerant in short time
	if elapsed = now.Sub(counter.failedAt); elapsed < counter.config.Threshold {
		return
	}

	// become unhealthy
	if counter.reports == 0 {
		unhealthy = true
		counter.reports++
		return
	}

	// remind time not reached
	if remind := counter.config.Threshold + counter.config.Remind*time.Duration(counter.reports); elapsed < remind {
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
func (counter *TimedCounter) OnError(err error) (recovered bool, unhealthy bool, unrecovered bool, elapsed time.Duration) {
	if isErrorNil(err) {
		recovered, elapsed = counter.OnSuccess()
	} else {
		unhealthy, unrecovered, elapsed = counter.OnFailure()
	}

	return
}
