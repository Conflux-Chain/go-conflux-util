package rate

import (
	"math"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// maxTokenBucketTimeoutSecs is the largest whole-second timeout that fits in time.Duration.
const maxTokenBucketTimeoutSecs = int64(math.MaxInt64) / int64(time.Second)

// TokenBucket limits requests using a token bucket. It is safe for concurrent
// use and must not be copied after first use.
type TokenBucket struct {
	inner   *rate.Limiter
	timeout time.Duration

	mu         sync.Mutex // protects the reservation/cancellation sequence and lastSeenAt
	lastSeenAt time.Time
}

// NewTokenBucket creates a token bucket with an integer refill rate.
// It panics if qps or burst is not positive.
// See NewTokenBucketRate for the automatic timeout's representable range.
func NewTokenBucket(qps int, burst int) Limiter {
	return NewTokenBucketRate(rate.Limit(qps), burst)
}

// NewTokenBucketRate creates a token bucket with an automatically calculated timeout.
// It panics if qps is not positive and finite, or burst is not positive.
// It also panics if the calculated whole-second timeout cannot fit in time.Duration.
// The finite rate.Inf constant is supported for unlimited consumption.
func NewTokenBucketRate(qps rate.Limit, burst int) Limiter {
	var timeoutSecs int64 = 10
	if qps > 0 && burst > 0 {
		refillSecs := float64(burst) / float64(qps)
		// Check before conversion and leave room for the extra second.
		if refillSecs >= float64(maxTokenBucketTimeoutSecs) {
			panic("rate: automatic token bucket timeout exceeds time.Duration range")
		}
		timeoutSecs = int64(refillSecs) + 1
	}

	return NewTokenBucketWithTimeout(float64(qps), burst, timeoutSecs)
}

// NewTokenBucketWithTimeout creates a token bucket with an explicit timeout in seconds.
// It panics if qps is not positive and finite, or burst or timeoutSecs is not positive.
// It also panics if timeoutSecs cannot fit in time.Duration (about 292 years).
// The full-bucket refill time (burst/qps) must not exceed the largest whole-second
// duration either, so the underlying limiter can represent reservation delays.
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

	if timeoutSecs > maxTokenBucketTimeoutSecs {
		panic("rate: token bucket timeout exceeds time.Duration range")
	}

	if float64(burst)/qps > float64(maxTokenBucketTimeoutSecs) {
		panic("rate: token bucket refill duration exceeds time.Duration range")
	}

	return &TokenBucket{
		inner:      rate.NewLimiter(rate.Limit(qps), burst),
		lastSeenAt: time.Now(),
		timeout:    time.Duration(timeoutSecs) * time.Second,
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

	// ReserveN and CancelAt must not interleave with another request's reservation.
	// Otherwise rejected requests can leave token debt behind after cancellation.
	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	rsv := bucket.inner.ReserveN(now, n)
	if !rsv.OK() {
		return errMaxExceeded(bucket.inner.Burst())
	}

	// Keep delay checks and cancellation on the same clock as ReserveN.
	if waitTime := rsv.DelayFrom(now); waitTime > 0 {
		rsv.CancelAt(now)
		return errRateLimited(bucket.inner.Limit(), waitTime)
	}

	bucket.lastSeenAt = now

	return nil
}

func (bucket *TokenBucket) Expired() bool {
	return bucket.ExpiredAt(time.Now())
}

// ExpiredAt reports whether now is strictly past the last successful request's
// time plus the timeout, or creation time plus the timeout if no request succeeded.
// It preserves full time precision; the exact deadline is not considered expired.
func (bucket *TokenBucket) ExpiredAt(now time.Time) bool {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	return bucket.lastSeenAt.Add(bucket.timeout).Before(now)
}
