package rate

import (
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
