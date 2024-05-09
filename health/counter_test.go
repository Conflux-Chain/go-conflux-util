package health

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestCounter() *Counter {
	return NewCounter(CounterConfig{
		Threshold: 5,
		Remind:    10,
	})
}

func TestCounterContinousSuccess(t *testing.T) {
	counter := newTestCounter()

	recovered, failures := counter.OnSuccess()
	assert.False(t, recovered)
	assert.Equal(t, uint64(0), failures)
}

func TestCounterFailedShortTime(t *testing.T) {
	counter := newTestCounter()

	// first failure
	unhealthy, unrecovered, failures := counter.OnFailure()
	assert.False(t, unhealthy)
	assert.False(t, unrecovered)
	assert.Equal(t, uint64(1), failures)

	// continous failure in short time
	unhealthy, unrecovered, failures = counter.OnFailure()
	assert.False(t, unhealthy)
	assert.False(t, unrecovered)
	assert.Equal(t, uint64(2), failures)

	// recovered
	recovered, failures := counter.OnSuccess()
	assert.False(t, recovered)
	assert.Equal(t, uint64(2), failures)
}

func TestCounterThreshold(t *testing.T) {
	counter := newTestCounter()

	// continous failure in short time
	for i := uint64(1); i <= counter.Threshold; i++ {
		unhealthy, unrecovered, failures := counter.OnFailure()
		assert.False(t, unhealthy)
		assert.False(t, unrecovered)
		assert.Equal(t, i, failures)

	}

	// continous failure in long time
	unhealthy, unrecovered, failures := counter.OnFailure()
	assert.True(t, unhealthy)
	assert.False(t, unrecovered)
	assert.Equal(t, counter.Threshold+1, failures)

	// continous failure in long time, but not reached to remind counter
	unhealthy, unrecovered, failures = counter.OnFailure()
	assert.False(t, unhealthy)
	assert.False(t, unrecovered)
	assert.Equal(t, counter.Threshold+2, failures)

	// recovered
	recovered, failures := counter.OnSuccess()
	assert.True(t, recovered)
	assert.Equal(t, counter.Threshold+2, failures)
}

func TestCounterRemind(t *testing.T) {
	counter := newTestCounter()

	// continous failure in short time
	for i := uint64(1); i <= counter.Threshold+counter.Remind; i++ {
		_, unrecovered, failures := counter.OnFailure()
		assert.False(t, unrecovered)
		assert.Equal(t, i, failures)
	}

	// continous failure and reached remind time
	unhealthy, unrecovered, failures := counter.OnFailure()
	assert.False(t, unhealthy)
	assert.True(t, unrecovered)
	assert.Equal(t, counter.Threshold+counter.Remind+1, failures)

	// recovered
	recovered, failures := counter.OnSuccess()
	assert.True(t, recovered)
	assert.Equal(t, counter.Threshold+counter.Remind+1, failures)
}
