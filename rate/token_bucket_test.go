package rate

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTokenBucketLimitAt(t *testing.T) {
	for _, tc := range []struct {
		name   string
		offset time.Duration
	}{
		{"past", -time.Hour},
		{"future", time.Hour},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limiter := NewTokenBucket(1, 1)
			now := time.Now().Add(tc.offset)

			// A full bucket allows one token at the supplied time.
			assert.NoError(t, limiter.LimitAt(now, 1))
			assert.ErrorContains(t, limiter.LimitAt(now, 1), "try again after 1s")
			assert.ErrorContains(t, limiter.LimitAt(now, 1), "try again after 1s")

			// Rejected reservations must not delay the next token's refill.
			next := now.Add(time.Second)
			assert.NoError(t, limiter.LimitAt(next, 1))
			assert.ErrorContains(t, limiter.LimitAt(next, 1), "try again after 1s")
		})
	}
}

func TestTokenBucketNonPositiveAmount(t *testing.T) {
	for _, tc := range []struct {
		name string
		n    int
	}{
		{"zero", 0},
		{"negative", -1},
		{"minimum int", math.MinInt},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limiter := NewTokenBucketWithTimeout(1, 1, 10)
			now := time.Now().Add(time.Hour).Truncate(time.Second)
			assert.NoError(t, limiter.LimitAt(now, 1))

			// Invalid amounts cannot refund the consumed token.
			assert.ErrorContains(t, limiter.LimitAt(now, tc.n), "must be positive")
			assert.Error(t, limiter.LimitAt(now, 1))

			// A later invalid request must not refresh the expiration time.
			assert.ErrorContains(t, limiter.LimitAt(now.Add(5*time.Second), tc.n), "must be positive")
			assert.True(t, limiter.ExpiredAt(now.Add(11*time.Second)))
		})
	}
}
