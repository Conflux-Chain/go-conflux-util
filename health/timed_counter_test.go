package health

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var testTimedCounterConfig = TimedCounterConfig{
	Threshold: time.Minute,
	Remind:    5 * time.Minute,
}

func TestTimedCounterContinuousSuccess(t *testing.T) {
	counter := NewTimedCounter(testTimedCounterConfig)

	recovered, elapsed := counter.onSuccessAt(time.Now().Add(testTimedCounterConfig.Threshold + 1))
	assert.False(t, recovered)
	assert.Equal(t, time.Duration(0), elapsed)
}

func TestTimedCounterFailedShortTime(t *testing.T) {
	counter := NewTimedCounter(testTimedCounterConfig)
	now := time.Now()

	// first failure
	unhealthy, unrecovered, elapsed := counter.onFailureAt(now)
	assert.False(t, unhealthy)
	assert.False(t, unrecovered)
	assert.Equal(t, time.Duration(0), elapsed)

	// continuous failure in short time
	unhealthy, unrecovered, elapsed = counter.onFailureAt(now.Add(testTimedCounterConfig.Threshold - 2))
	assert.False(t, unhealthy)
	assert.False(t, unrecovered)
	assert.Equal(t, testTimedCounterConfig.Threshold-2, elapsed)

	// recovered
	recovered, elapsed := counter.onSuccessAt(now.Add(testTimedCounterConfig.Threshold - 1))
	assert.False(t, recovered)
	assert.Equal(t, testTimedCounterConfig.Threshold-1, elapsed)
}

func TestTimedCounterThreshold(t *testing.T) {
	counter := NewTimedCounter(testTimedCounterConfig)
	now := time.Now()

	// first failure
	counter.onFailureAt(now)

	// continuous failure in short time
	counter.onFailureAt(now.Add(testTimedCounterConfig.Threshold - 1))

	// continuous failure in long time
	unhealthy, unrecovered, elapsed := counter.onFailureAt(now.Add(testTimedCounterConfig.Threshold + 1))
	assert.True(t, unhealthy)
	assert.False(t, unrecovered)
	assert.Equal(t, testTimedCounterConfig.Threshold+1, elapsed)

	// recovered
	recovered, elapsed := counter.onSuccessAt(now.Add(testTimedCounterConfig.Threshold + 2))
	assert.True(t, recovered)
	assert.Equal(t, testTimedCounterConfig.Threshold+2, elapsed)
}

func TestTimedCounterRemind(t *testing.T) {
	counter := NewTimedCounter(testTimedCounterConfig)
	now := time.Now()

	// first failure
	counter.onFailureAt(now)

	// continuous failure in short time
	counter.onFailureAt(now.Add(testTimedCounterConfig.Threshold - 1))

	// continuous failure in long time
	counter.onFailureAt(now.Add(testTimedCounterConfig.Threshold + 1))

	// continuous failure in long time, but not reached remind time
	unhealthy, unrecovered, elapsed := counter.onFailureAt(now.Add(testTimedCounterConfig.Threshold + 2))
	assert.False(t, unhealthy)
	assert.False(t, unrecovered)
	assert.Equal(t, testTimedCounterConfig.Threshold+2, elapsed)

	// continuous failure and reached remind time
	unhealthy, unrecovered, elapsed = counter.onFailureAt(now.Add(testTimedCounterConfig.Threshold + 2 + testTimedCounterConfig.Remind))
	assert.False(t, unhealthy)
	assert.True(t, unrecovered)
	assert.Equal(t, testTimedCounterConfig.Threshold+2+testTimedCounterConfig.Remind, elapsed)

	// recovered
	recovered, elapsed := counter.onSuccessAt(now.Add(testTimedCounterConfig.Threshold + 3 + testTimedCounterConfig.Remind))
	assert.True(t, recovered)
	assert.Equal(t, testTimedCounterConfig.Threshold+3+testTimedCounterConfig.Remind, elapsed)
}
