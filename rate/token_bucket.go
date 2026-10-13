package rate

import (
	"math"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

type TokenBucket struct {
	inner       *rate.Limiter
	lastSeen    int64
	timeoutSecs int64
}

// NewTokenBucket creates a token bucket with an integer refill rate.
// It panics if qps or burst is not positive.
// See NewTokenBucketRate for the automatic timeout's representable range.
func NewTokenBucket(qps int, burst int) Limiter {
	return NewTokenBucketRate(rate.Limit(qps), burst)
}

// NewTokenBucketRate creates a token bucket with an automatically calculated timeout.
// It panics if qps is not positive and finite, or burst is not positive.
// It also panics if the calculated timeout cannot fit in int64 seconds.
// The finite rate.Inf constant is supported for unlimited consumption.
func NewTokenBucketRate(qps rate.Limit, burst int) Limiter {
	var timeoutSecs int64 = 10
	if qps > 0 && burst > 0 {
		refillSecs := float64(burst) / float64(qps)
		// float64(MaxInt64) rounds up to 2^63. Smaller float64 values
		// leave enough room for the extra second after conversion.
		if refillSecs >= float64(math.MaxInt64) {
			panic("rate: automatic token bucket timeout exceeds int64 seconds")
		}
		timeoutSecs = int64(refillSecs) + 1
	}

	return NewTokenBucketWithTimeout(float64(qps), burst, timeoutSecs)
}

// NewTokenBucketWithTimeout creates a token bucket with an explicit timeout in seconds.
// It panics if qps is not positive and finite, or burst or timeoutSecs is not positive.
// The finite rate.Inf constant is supported for unlimited consumption.
func NewTokenBucketWithTimeout(qps float64, burst int, timeoutSecs int64) Limiter {
	if qps <= 0 || math.IsNaN(qps) || math.IsInf(qps, 0) {
		panic("rate: token bucket qps must be positive and finite")
	}

	if burst <= 0 {
		panic("rate: token bucket burst must be positive")
	}

	if timeoutSecs <= 0 {
		panic("rate: token bucket timeout must be positive")
	}

	return &TokenBucket{
		inner:       rate.NewLimiter(rate.Limit(qps), burst),
		lastSeen:    time.Now().Unix(),
		timeoutSecs: timeoutSecs,
	}
}

func (bucket *TokenBucket) Limit() error {
	return bucket.LimitAt(time.Now(), 1)
}

// LimitN consumes n tokens at the current time, with the same amount rules as LimitAt.
func (bucket *TokenBucket) LimitN(n int) error {
	return bucket.LimitAt(time.Now(), n)
}

// LimitAt consumes n tokens at now. The amount must be positive;
// otherwise it returns an error without changing state.
func (bucket *TokenBucket) LimitAt(now time.Time, n int) error {
	if n <= 0 {
		return errInvalidAmount(n)
	}

	rsv := bucket.inner.ReserveN(now, n)
	if !rsv.OK() {
		return errMaxExceeded(bucket.inner.Burst())
	}

	// Keep delay checks and cancellation on the same clock as ReserveN.
	if waitTime := rsv.DelayFrom(now); waitTime > 0 {
		rsv.CancelAt(now)
		return errRateLimited(int(bucket.inner.Limit()), waitTime)
	}

	atomic.StoreInt64(&bucket.lastSeen, now.Unix())

	return nil
}

func (bucket *TokenBucket) Expired() bool {
	return bucket.ExpiredAt(time.Now())
}

// ExpiredAt reports whether now is past the expiration time, in Unix seconds.
// It assumes ordinary business timestamps and timeout values whose sum fits in
// int64 Unix seconds; overflow from extreme values is not handled.
func (bucket *TokenBucket) ExpiredAt(now time.Time) bool {
	lastSeen := atomic.LoadInt64(&bucket.lastSeen)
	return now.Unix() > lastSeen+bucket.timeoutSecs
}
