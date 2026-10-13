package rate

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	xrate "golang.org/x/time/rate"
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

func TestNewTokenBucketInvalidConfig(t *testing.T) {
	for _, tc := range []struct {
		name string
		new  func() Limiter
	}{
		{"integer zero qps", func() Limiter { return NewTokenBucket(0, 1) }},
		{"integer negative qps", func() Limiter { return NewTokenBucket(-1, 1) }},
		{"integer zero burst", func() Limiter { return NewTokenBucket(1, 0) }},
		{"integer negative burst", func() Limiter { return NewTokenBucket(1, -1) }},
		{"rate zero qps", func() Limiter { return NewTokenBucketRate(0, 1) }},
		{"rate negative qps", func() Limiter { return NewTokenBucketRate(-1, 1) }},
		{"rate NaN qps", func() Limiter { return NewTokenBucketRate(xrate.Limit(math.NaN()), 1) }},
		{"rate infinite qps", func() Limiter { return NewTokenBucketRate(xrate.Limit(math.Inf(1)), 1) }},
		{"rate negative infinite qps", func() Limiter { return NewTokenBucketRate(xrate.Limit(math.Inf(-1)), 1) }},
		{"rate zero burst", func() Limiter { return NewTokenBucketRate(1, 0) }},
		{"rate negative burst", func() Limiter { return NewTokenBucketRate(1, -1) }},
		{"timeout zero qps", func() Limiter { return NewTokenBucketWithTimeout(0, 1, 10) }},
		{"timeout negative qps", func() Limiter { return NewTokenBucketWithTimeout(-1, 1, 10) }},
		{"timeout NaN qps", func() Limiter { return NewTokenBucketWithTimeout(math.NaN(), 1, 10) }},
		{"timeout infinite qps", func() Limiter { return NewTokenBucketWithTimeout(math.Inf(1), 1, 10) }},
		{"timeout negative infinite qps", func() Limiter { return NewTokenBucketWithTimeout(math.Inf(-1), 1, 10) }},
		{"timeout zero burst", func() Limiter { return NewTokenBucketWithTimeout(1, 0, 10) }},
		{"timeout negative burst", func() Limiter { return NewTokenBucketWithTimeout(1, -1, 10) }},
		{"zero timeout", func() Limiter { return NewTokenBucketWithTimeout(1, 1, 0) }},
		{"negative timeout", func() Limiter { return NewTokenBucketWithTimeout(1, 1, -1) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Panics(t, func() { tc.new() })
		})
	}
}

func TestNewTokenBucketValidConfig(t *testing.T) {
	for _, tc := range []struct {
		name string
		new  func() Limiter
		n    int
	}{
		{"integer rate", func() Limiter { return NewTokenBucket(1, 1) }, 1},
		{"fractional rate", func() Limiter { return NewTokenBucketRate(0.5, 1) }, 1},
		{"explicit timeout", func() Limiter { return NewTokenBucketWithTimeout(0.5, 1, 10) }, 1},
		{"unlimited rate", func() Limiter { return NewTokenBucketRate(xrate.Inf, 1) }, 2},
		{"unlimited with timeout", func() Limiter { return NewTokenBucketWithTimeout(float64(xrate.Inf), 1, 10) }, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limiter := tc.new()
			now := time.Now()
			// rate.Inf continues to allow consumption beyond the burst size.
			assert.NoError(t, limiter.LimitAt(now, tc.n))
			assert.False(t, limiter.ExpiredAt(now))
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
