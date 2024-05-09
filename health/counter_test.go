package health

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var testCounterConfig = CounterConfig{
	Threshold: 5,
	Remind:    10,
}

func TestCounterContinousSuccess(t *testing.T) {
	var counter Counter

	recovered, failures := counter.OnSuccess(testCounterConfig)
	assert.False(t, recovered)
	assert.Equal(t, uint64(0), failures)
}

func TestCounterFailedShortTime(t *testing.T) {
	var counter Counter

	// first failure
	unhealthy, unrecovered, failures := counter.OnFailure(testCounterConfig)
	assert.False(t, unhealthy)
	assert.False(t, unrecovered)
	assert.Equal(t, uint64(1), failures)

	// continous failure in short time
	unhealthy, unrecovered, failures = counter.OnFailure(testCounterConfig)
	assert.False(t, unhealthy)
	assert.False(t, unrecovered)
	assert.Equal(t, uint64(2), failures)

	// recovered
	recovered, failures := counter.OnSuccess(testCounterConfig)
	assert.False(t, recovered)
	assert.Equal(t, uint64(2), failures)
}

func TestCounterThreshold(t *testing.T) {
	var counter Counter

	// continous failure in short time
	for i := uint64(1); i <= testCounterConfig.Threshold; i++ {
		unhealthy, unrecovered, failures := counter.OnFailure(testCounterConfig)
		assert.False(t, unhealthy)
		assert.False(t, unrecovered)
		assert.Equal(t, i, failures)

	}

	// continous failure in long time
	unhealthy, unrecovered, failures := counter.OnFailure(testCounterConfig)
	assert.True(t, unhealthy)
	assert.False(t, unrecovered)
	assert.Equal(t, testCounterConfig.Threshold+1, failures)

	// continous failure in long time, but not reached to remind counter
	unhealthy, unrecovered, failures = counter.OnFailure(testCounterConfig)
	assert.False(t, unhealthy)
	assert.False(t, unrecovered)
	assert.Equal(t, testCounterConfig.Threshold+2, failures)

	// recovered
	recovered, failures := counter.OnSuccess(testCounterConfig)
	assert.True(t, recovered)
	assert.Equal(t, testCounterConfig.Threshold+2, failures)
}

func TestCounterRemind(t *testing.T) {
	var counter Counter

	// continous failure in short time
	for i := uint64(1); i <= testCounterConfig.Threshold+testCounterConfig.Remind; i++ {
		_, unrecovered, failures := counter.OnFailure(testCounterConfig)
		assert.False(t, unrecovered)
		assert.Equal(t, i, failures)
	}

	// continous failure and reached remind time
	unhealthy, unrecovered, failures := counter.OnFailure(testCounterConfig)
	assert.False(t, unhealthy)
	assert.True(t, unrecovered)
	assert.Equal(t, testCounterConfig.Threshold+testCounterConfig.Remind+1, failures)

	// recovered
	recovered, failures := counter.OnSuccess(testCounterConfig)
	assert.True(t, recovered)
	assert.Equal(t, testCounterConfig.Threshold+testCounterConfig.Remind+1, failures)
}
