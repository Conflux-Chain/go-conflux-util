package health

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var testCounterConfig = CounterConfig{
	Threshold: 5,
	Remind:    10,
}

func TestCounterContinuousSuccess(t *testing.T) {
	counter := NewCounter(testCounterConfig)

	recovered, failures := counter.OnSuccess()
	assert.False(t, recovered)
	assert.Equal(t, uint64(0), failures)
}

func TestCounterFailedShortTime(t *testing.T) {
	counter := NewCounter(testCounterConfig)

	// first failure
	unhealthy, unrecovered, failures := counter.OnFailure()
	assert.False(t, unhealthy)
	assert.False(t, unrecovered)
	assert.Equal(t, uint64(1), failures)

	// continuous failure in short time
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
	counter := NewCounter(testCounterConfig)

	// continuous failure in short time
	for i := uint64(1); i < testCounterConfig.Threshold; i++ {
		unhealthy, unrecovered, failures := counter.OnFailure()
		assert.False(t, unhealthy)
		assert.False(t, unrecovered)
		assert.Equal(t, i, failures)

	}

	// continuous failure in long time
	unhealthy, unrecovered, failures := counter.OnFailure()
	assert.True(t, unhealthy)
	assert.False(t, unrecovered)
	assert.Equal(t, testCounterConfig.Threshold, failures)

	// continuous failure in long time, but not reached to remind counter
	unhealthy, unrecovered, failures = counter.OnFailure()
	assert.False(t, unhealthy)
	assert.False(t, unrecovered)
	assert.Equal(t, testCounterConfig.Threshold+1, failures)

	// recovered
	recovered, failures := counter.OnSuccess()
	assert.True(t, recovered)
	assert.Equal(t, testCounterConfig.Threshold+1, failures)
}

func TestCounterRemind(t *testing.T) {
	counter := NewCounter(testCounterConfig)

	// continuous failure in short time
	for i := uint64(1); i < testCounterConfig.Threshold+testCounterConfig.Remind; i++ {
		_, unrecovered, failures := counter.OnFailure()
		assert.False(t, unrecovered)
		assert.Equal(t, i, failures)
	}

	// continuous failure and reached remind time
	unhealthy, unrecovered, failures := counter.OnFailure()
	assert.False(t, unhealthy)
	assert.True(t, unrecovered)
	assert.Equal(t, testCounterConfig.Threshold+testCounterConfig.Remind, failures)

	// recovered
	recovered, failures := counter.OnSuccess()
	assert.True(t, recovered)
	assert.Equal(t, testCounterConfig.Threshold+testCounterConfig.Remind, failures)
}
