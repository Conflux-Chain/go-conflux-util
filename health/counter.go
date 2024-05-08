package health

type CounterConfig struct {
	Threshold uint64 `default:"60"` // report unhealthy if threshold reached
	Remind    uint64 `default:"60"` // remind unhealthy if unrecovered for a long time
}

// Counter represents an error tolerant health counter, which allows failures in short time
// and periodically remind unhealthy if unrecovered in time.
type Counter struct {
	CounterConfig

	failures uint64
}

func NewCounter(config CounterConfig) *Counter {
	return &Counter{
		CounterConfig: config,
	}
}

// IsSuccess indicates whether any failure occurred.
func (counter *Counter) IsSuccess() bool {
	return counter.failures == 0
}

// OnSuccess erases failure status and return recover information if any.
//
// `recovered`: indicates if recovered from unhealthy status.
//
// `failures`: indicates the number of failures before success.
func (counter *Counter) OnSuccess() (recovered bool, failures uint64) {
	// last time was success status
	if counter.failures == 0 {
		return
	}

	// report health now after a long time
	if failures = counter.failures; failures > counter.Threshold {
		recovered = true
	}

	// reset
	counter.failures = 0

	return
}

// OnFailure marks failure status and return unhealthy information.
//
// `unhealthy`: indicates continous failures in a long time.
//
// `unrecovered`: indicates continous failures and unrecovered in a long time.
//
// `failures`: indicates the number of failures so far.
func (counter *Counter) OnFailure() (unhealthy bool, unrecovered bool, failures uint64) {
	counter.failures++

	// error tolerant in short time
	if failures = counter.failures; failures <= counter.Threshold {
		return
	}

	if delta := failures - counter.Threshold - 1; delta == 0 {
		unhealthy = true
	} else if delta%counter.Remind == 0 {
		unrecovered = true
	}

	return
}
