package rate

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFixedWindow(t *testing.T) {
	limiter := NewFixedWindow(time.Minute, 10)
	now := time.Now().Truncate(time.Minute).Add(time.Minute)

	// Rejected requests do not consume the remaining quota.
	assert.NoError(t, limiter.LimitAt(now, 1))
	assert.Error(t, limiter.LimitAt(now, 11))
	assert.Error(t, limiter.LimitAt(now, 10))
	assert.NoError(t, limiter.LimitAt(now, 9))

	// A full window rejects requests until the next boundary.
	next := now.Add(time.Minute)
	assert.Error(t, limiter.LimitAt(now, 1))
	assert.Error(t, limiter.LimitAt(next.Add(-time.Nanosecond), 1))
	assert.False(t, limiter.ExpiredAt(next))
	assert.True(t, limiter.ExpiredAt(next.Add(time.Nanosecond)))

	// Crossing the boundary resets the quota and advances expiration.
	assert.NoError(t, limiter.LimitAt(next, 1))
	assert.NoError(t, limiter.LimitAt(next, 9))
	assert.Error(t, limiter.LimitAt(next, 1))
	assert.False(t, limiter.ExpiredAt(next.Add(time.Nanosecond)))
}

func TestFixedWindowNonPositiveMax(t *testing.T) {
	for _, tc := range []struct {
		name string
		max  int
	}{
		{"zero", 0},
		{"negative", -1},
		{"minimum int", math.MinInt},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limiter := NewFixedWindow(time.Minute, tc.max)
			now := time.Now().Truncate(time.Minute).Add(time.Minute)
			assert.ErrorContains(t, limiter.LimitAt(now, 0), "must be positive")
			assert.ErrorContains(t, limiter.LimitAt(now, -1), "must be positive")
			assert.Error(t, limiter.LimitAt(now, 1))
			assert.Error(t, limiter.LimitAt(now.Add(time.Minute), 1))
		})
	}
}

func TestFixedWindowNonPositiveAmount(t *testing.T) {
	for _, tc := range []struct {
		name string
		n    int
	}{
		{"zero", 0},
		{"negative", -1},
		{"minimum int", math.MinInt},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limiter := NewFixedWindow(time.Minute, 1)
			now := time.Now().Truncate(time.Minute).Add(time.Minute)
			assert.NoError(t, limiter.LimitAt(now, 1))

			assert.ErrorContains(t, limiter.LimitAt(now.Add(time.Hour), tc.n), "must be positive")

			// Invalid amounts do not advance the window or refund quota.
			assert.True(t, limiter.ExpiredAt(now.Add(time.Minute+time.Nanosecond)))
			assert.Error(t, limiter.LimitAt(now, 1))
		})
	}
}

func TestFixedWindowMaxIntQuota(t *testing.T) {
	limiter := NewFixedWindow(time.Minute, math.MaxInt)
	now := time.Now().Truncate(time.Minute).Add(time.Minute)
	assert.NoError(t, limiter.LimitAt(now, math.MaxInt-1))
	assert.Error(t, limiter.LimitAt(now, 2))
	assert.NoError(t, limiter.LimitAt(now, 1))
	assert.Error(t, limiter.LimitAt(now, 1))
}

func TestFixedWindowSubMillisecondInterval(t *testing.T) {
	interval := 500 * time.Microsecond
	limiter := NewFixedWindow(interval, 1)
	now := time.Now().Truncate(time.Millisecond).Add(time.Second)
	assert.NoError(t, limiter.LimitAt(now, 1))
	assert.Error(t, limiter.LimitAt(now.Add(interval-time.Nanosecond), 1))

	next := now.Add(interval)
	assert.NoError(t, limiter.LimitAt(next, 1))
	assert.Error(t, limiter.LimitAt(next, 1))
	assert.False(t, limiter.ExpiredAt(next.Add(time.Nanosecond)))
	assert.True(t, limiter.ExpiredAt(next.Add(interval+time.Nanosecond)))
}
