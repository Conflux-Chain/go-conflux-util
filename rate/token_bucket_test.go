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

func TestValidateTokenBucketConfig(t *testing.T) {
	for _, tc := range []struct {
		name        string
		qps         float64
		burst       int
		timeoutSecs int64
		err         string
	}{
		{"valid", 1, 1, 60, ""},
		{"fractional rate", 0.5, 1, 60, ""},
		{"unlimited rate", float64(xrate.Inf), 1, 60, ""},
		{"maximum timeout", 1, 1, maxTokenBucketTimeoutSecs, ""},
		{"maximum refill duration", 1 / float64(maxTokenBucketTimeoutSecs), 1, 60, ""},
		{"zero rate", 0, 1, 60, "qps must be positive and finite"},
		{"NaN rate", math.NaN(), 1, 60, "qps must be positive and finite"},
		{"infinite rate", math.Inf(1), 1, 60, "qps must be positive and finite"},
		{"zero burst", 1, 0, 60, "burst must be positive"},
		{"zero timeout", 1, 1, 0, "timeout must be positive"},
		{"timeout overflow", 1, 1, maxTokenBucketTimeoutSecs + 1, "timeout exceeds time.Duration range"},
		{"refill duration overflow", 1e-10, 1, 60, "refill duration exceeds time.Duration range"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTokenBucketConfig(tc.qps, tc.burst, tc.timeoutSecs)
			if tc.err == "" {
				assert.NoError(t, err)
				return
			}

			if assert.ErrorContains(t, err, tc.err) {
				// Constructors retain string panic values; validation returns errors.
				assert.PanicsWithValue(t, err.Error(), func() {
					NewTokenBucketWithTimeout(tc.qps, tc.burst, tc.timeoutSecs)
				})
			}
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
		{"timeout exceeds duration", func() Limiter { return NewTokenBucketWithTimeout(1, 1, maxTokenBucketTimeoutSecs+1) }},
		{"maximum int64 timeout", func() Limiter { return NewTokenBucketWithTimeout(1, 1, math.MaxInt64) }},
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
		{"maximum timeout", func() Limiter { return NewTokenBucketWithTimeout(1, 1, maxTokenBucketTimeoutSecs) }, 1},
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

func TestTokenBucketAutomaticTimeoutRange(t *testing.T) {
	for _, tc := range []struct {
		name string
		qps  xrate.Limit
	}{
		{"duration boundary", xrate.Limit(math.Nextafter(1/float64(maxTokenBucketTimeoutSecs), 0))},
		{"beyond duration", xrate.Limit(1 / float64(maxTokenBucketTimeoutSecs+1))},
		{"int64 boundary", xrate.Limit(1 / float64(math.MaxInt64))},
		{"beyond int64", 1e-20},
		{"infinite quotient", xrate.Limit(math.SmallestNonzeroFloat64)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.PanicsWithValue(t, "rate: automatic token bucket timeout exceeds time.Duration range", func() {
				NewTokenBucketRate(tc.qps, 1)
			})
		})
	}

	// A slightly larger rate makes the computed timeout fit below the boundary.
	qps := math.Nextafter(1/float64(maxTokenBucketTimeoutSecs), math.Inf(1))
	limiter := NewTokenBucketRate(xrate.Limit(qps), 1)
	now := time.Unix(1700000000, 0)
	assert.NoError(t, limiter.LimitAt(now, 1))
	deadline := now.Add(time.Duration(maxTokenBucketTimeoutSecs) * time.Second)
	assert.False(t, limiter.ExpiredAt(deadline))
	assert.True(t, limiter.ExpiredAt(deadline.Add(time.Second)))
}

func TestTokenBucketExpirationBoundary(t *testing.T) {
	for _, tc := range []struct {
		name    string
		qps     xrate.Limit
		timeout time.Duration
	}{
		{"integer rate", 1, 2 * time.Second},
		{"fractional rate", 0.4, 3 * time.Second},
		{"unlimited rate", xrate.Inf, time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limiter := NewTokenBucketRate(tc.qps, 1)
			now := time.Unix(1700000000, int64(500*time.Millisecond))
			assert.NoError(t, limiter.LimitAt(now, 1))
			deadline := now.Add(tc.timeout)
			assert.False(t, limiter.ExpiredAt(deadline.Add(-time.Nanosecond)))
			assert.False(t, limiter.ExpiredAt(deadline))
			assert.True(t, limiter.ExpiredAt(deadline.Add(time.Nanosecond)))
		})
	}
}

func TestTokenBucketRefillDurationRange(t *testing.T) {
	for _, tc := range []struct {
		name  string
		qps   float64
		burst int
	}{
		{"beyond boundary", math.Nextafter(1/float64(maxTokenBucketTimeoutSecs), 0), 1},
		{"slow single token", 1e-10, 1},
		{"slow full burst", 1e-9, 10},
		{"infinite quotient", math.SmallestNonzeroFloat64, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.PanicsWithValue(t, "rate: token bucket refill duration exceeds time.Duration range", func() {
				NewTokenBucketWithTimeout(tc.qps, tc.burst, 60)
			})
		})
	}

	// A refill at the largest supported whole-second duration is still valid.
	limiter := NewTokenBucketWithTimeout(1/float64(maxTokenBucketTimeoutSecs), 1, 60)
	now := time.Unix(1700000000, 0)
	assert.NoError(t, limiter.LimitAt(now, 1))
	assert.ErrorIs(t, limiter.LimitAt(now, 1), ErrRateLimited)
}

func TestTokenBucketSlowRefill(t *testing.T) {
	limiter := NewTokenBucketWithTimeout(1e-9, 1, 60)
	now := time.Unix(1700000000, 0)
	assert.NoError(t, limiter.LimitAt(now, 1))
	assert.ErrorIs(t, limiter.LimitAt(now, 1), ErrRateLimited)
	assert.ErrorIs(t, limiter.LimitAt(now, 1), ErrRateLimited)
	// Rejected reservations must not consume the next token, even at a low rate.
	assert.NoError(t, limiter.LimitAt(now.Add(1_000_000_000*time.Second), 1))
}

func TestTokenBucketFractionalRateError(t *testing.T) {
	for _, tc := range []struct {
		name    string
		qps     xrate.Limit
		message string
	}{
		{"below one", 0.5, "Too many requests (exceeds 0.5), try again after 2s"},
		{"above one", 1.5, "Too many requests (exceeds 1.5), try again after 667ms"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limiter := NewTokenBucketRate(tc.qps, 1)
			now := time.Unix(1700000000, 0)
			assert.NoError(t, limiter.LimitAt(now, 1))
			err := limiter.LimitAt(now, 1)
			assert.EqualError(t, err, tc.message)
			assert.ErrorIs(t, err, ErrRateLimited)
		})
	}
}
