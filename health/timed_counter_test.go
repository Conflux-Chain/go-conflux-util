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

	// one nanosecond before the threshold
	unhealthy, unrecovered, elapsed := counter.onFailureAt(now.Add(testTimedCounterConfig.Threshold - time.Nanosecond))
	assert.False(t, unhealthy)
	assert.False(t, unrecovered)
	assert.Equal(t, testTimedCounterConfig.Threshold-time.Nanosecond, elapsed)

	// exactly at the threshold
	unhealthy, unrecovered, elapsed = counter.onFailureAt(now.Add(testTimedCounterConfig.Threshold))
	assert.True(t, unhealthy)
	assert.False(t, unrecovered)
	assert.Equal(t, testTimedCounterConfig.Threshold, elapsed)

	// recovery exactly at the threshold
	recovered, elapsed := counter.onSuccessAt(now.Add(testTimedCounterConfig.Threshold))
	assert.True(t, recovered)
	assert.Equal(t, testTimedCounterConfig.Threshold, elapsed)
	assert.True(t, counter.IsSuccess())
}

func TestTimedCounterRemind(t *testing.T) {
	counter := NewTimedCounter(testTimedCounterConfig)
	now := time.Now()

	// first failure
	counter.onFailureAt(now)

	// exactly at the threshold
	unhealthy, unrecovered, elapsed := counter.onFailureAt(now.Add(testTimedCounterConfig.Threshold))
	assert.True(t, unhealthy)
	assert.False(t, unrecovered)
	assert.Equal(t, testTimedCounterConfig.Threshold, elapsed)

	// one nanosecond before the reminder
	remindAt := testTimedCounterConfig.Threshold + testTimedCounterConfig.Remind
	unhealthy, unrecovered, elapsed = counter.onFailureAt(now.Add(remindAt - time.Nanosecond))
	assert.False(t, unhealthy)
	assert.False(t, unrecovered)
	assert.Equal(t, remindAt-time.Nanosecond, elapsed)

	// exactly at the reminder
	unhealthy, unrecovered, elapsed = counter.onFailureAt(now.Add(remindAt))
	assert.False(t, unhealthy)
	assert.True(t, unrecovered)
	assert.Equal(t, remindAt, elapsed)

	// recovered
	recovered, elapsed := counter.onSuccessAt(now.Add(remindAt))
	assert.True(t, recovered)
	assert.Equal(t, remindAt, elapsed)
	assert.True(t, counter.IsSuccess())

	// a new failure starts a fresh interval after recovery
	restartedAt := now.Add(remindAt + time.Second)
	unhealthy, unrecovered, elapsed = counter.onFailureAt(restartedAt)
	assert.False(t, unhealthy)
	assert.False(t, unrecovered)
	assert.Equal(t, time.Duration(0), elapsed)
	assert.False(t, counter.IsSuccess())

	// the new interval reports unhealthy again at its own threshold
	unhealthy, unrecovered, elapsed = counter.onFailureAt(restartedAt.Add(testTimedCounterConfig.Threshold))
	assert.True(t, unhealthy)
	assert.False(t, unrecovered)
	assert.Equal(t, testTimedCounterConfig.Threshold, elapsed)
}
