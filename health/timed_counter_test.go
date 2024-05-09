package health

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func newTestTimedCounter() *TimedCounter {
	return NewTimedCounter(TimedCounterConfig{
		Threshold: time.Minute,
		Remind:    5 * time.Minute,
	})
}

func TestTimedCounterContinousSuccess(t *testing.T) {
	counter := newTestTimedCounter()

	recovered, elapsed := counter.onSuccessAt(time.Now().Add(counter.Threshold + 1))
	assert.False(t, recovered)
	assert.Equal(t, time.Duration(0), elapsed)
}

func TestTimedCounterFailedShortTime(t *testing.T) {
	counter := newTestTimedCounter()
	now := time.Now()

	// first failure
	unhealthy, unrecovered, elapsed := counter.onFailureAt(now)
	assert.False(t, unhealthy)
	assert.False(t, unrecovered)
	assert.Equal(t, time.Duration(0), elapsed)

	// continous failure in short time
	unhealthy, unrecovered, elapsed = counter.onFailureAt(now.Add(counter.Threshold - 2))
	assert.False(t, unhealthy)
	assert.False(t, unrecovered)
	assert.Equal(t, counter.Threshold-2, elapsed)

	// recovered
	recovered, elapsed := counter.onSuccessAt(now.Add(counter.Threshold - 1))
	assert.False(t, recovered)
	assert.Equal(t, counter.Threshold-1, elapsed)
}

func TestTimedCounterThreshold(t *testing.T) {
	counter := newTestTimedCounter()
	now := time.Now()

	// first failure
	counter.onFailureAt(now)

	// continous failure in short time
	counter.onFailureAt(now.Add(counter.Threshold - 1))

	// continous failure in long time
	unhealthy, unrecovered, elapsed := counter.onFailureAt(now.Add(counter.Threshold + 1))
	assert.True(t, unhealthy)
	assert.False(t, unrecovered)
	assert.Equal(t, counter.Threshold+1, elapsed)

	// recovered
	recovered, elapsed := counter.onSuccessAt(now.Add(counter.Threshold + 2))
	assert.True(t, recovered)
	assert.Equal(t, counter.Threshold+2, elapsed)
}

func TestTimedCounterRemind(t *testing.T) {
	counter := newTestTimedCounter()
	now := time.Now()

	// first failure
	counter.onFailureAt(now)

	// continous failure in short time
	counter.onFailureAt(now.Add(counter.Threshold - 1))

	// continous failure in long time
	counter.onFailureAt(now.Add(counter.Threshold + 1))

	// continous failure in long time, but not reached remind time
	unhealthy, unrecovered, elapsed := counter.onFailureAt(now.Add(counter.Threshold + 2))
	assert.False(t, unhealthy)
	assert.False(t, unrecovered)
	assert.Equal(t, counter.Threshold+2, elapsed)

	// continous failure and reached remind time
	unhealthy, unrecovered, elapsed = counter.onFailureAt(now.Add(counter.Threshold + 2 + counter.Remind))
	assert.False(t, unhealthy)
	assert.True(t, unrecovered)
	assert.Equal(t, counter.Threshold+2+counter.Remind, elapsed)

	// recovered
	recovered, elapsed := counter.onSuccessAt(now.Add(counter.Threshold + 3 + counter.Remind))
	assert.True(t, recovered)
	assert.Equal(t, counter.Threshold+3+counter.Remind, elapsed)
}
