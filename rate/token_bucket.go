package rate

import (
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

type TokenBucket struct {
	inner       *rate.Limiter
	lastSeen    int64
	timeoutSecs int64
}

func NewTokenBucket(qps int, burst int) Limiter {
	return NewTokenBucketRate(rate.Limit(qps), burst)
}

func NewTokenBucketRate(qps rate.Limit, burst int) Limiter {
	var timeoutSecs int64 = 10
	if qps > 0 {
		timeoutSecs = int64(float64(burst)/float64(qps)) + 1
	}

	return NewTokenBucketWithTimeout(float64(qps), burst, timeoutSecs)
}

func NewTokenBucketWithTimeout(qps float64, burst int, timeoutSecs int64) Limiter {
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

func (bucket *TokenBucket) ExpiredAt(now time.Time) bool {
	lastSeen := atomic.LoadInt64(&bucket.lastSeen)
	return now.Unix() > lastSeen+bucket.timeoutSecs
}
