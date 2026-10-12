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

func TestNewTimedCounterAppliesDefaultsToExplicitConfig(t *testing.T) {
	tests := []struct {
		name   string
		config TimedCounterConfig
		want   TimedCounterConfig
	}{
		{name: "zero config", config: TimedCounterConfig{}, want: DefaultTimedCounterConfig()},
		{name: "partial config", config: TimedCounterConfig{Threshold: 3 * time.Minute}, want: TimedCounterConfig{
			Threshold: 3 * time.Minute,
			Remind:    5 * time.Minute,
		}},
		{name: "negative threshold", config: TimedCounterConfig{Threshold: -time.Second, Remind: 2 * time.Minute}, want: TimedCounterConfig{
			Threshold: time.Minute,
			Remind:    2 * time.Minute,
		}},
		{name: "negative remind", config: TimedCounterConfig{Threshold: 3 * time.Minute, Remind: -time.Second}, want: TimedCounterConfig{
			Threshold: 3 * time.Minute,
			Remind:    5 * time.Minute,
		}},
		{name: "both negative", config: TimedCounterConfig{Threshold: -time.Second, Remind: -time.Second}, want: DefaultTimedCounterConfig()},
		{name: "complete config", config: testTimedCounterConfig, want: testTimedCounterConfig},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := NewTimedCounter(tt.config)
			assert.Equal(t, tt.want, counter.config)
		})
	}
}

func TestTimedCounterOnError(t *testing.T) {
	counter := NewTimedCounter(testTimedCounterConfig)
	// Record a past failure to reach the threshold without sleeping.
	counter.onFailureAt(time.Now().Add(-testTimedCounterConfig.Threshold))

	recovered, unhealthy, unrecovered, elapsed := counter.OnError(assert.AnError)
	assert.False(t, recovered)
	assert.True(t, unhealthy)
	assert.False(t, unrecovered)
	assert.GreaterOrEqual(t, elapsed, testTimedCounterConfig.Threshold)

	recovered, unhealthy, unrecovered, elapsed = counter.OnError(nil)
	assert.True(t, recovered)
	assert.False(t, unhealthy)
	assert.False(t, unrecovered)
	assert.GreaterOrEqual(t, elapsed, testTimedCounterConfig.Threshold)
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
