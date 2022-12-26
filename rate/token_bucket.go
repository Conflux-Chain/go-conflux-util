package rate

import (
	"time"

	"golang.org/x/time/rate"
)

type TokenBucket struct {
	inner *rate.Limiter
}

func NewTokenBucket(qps int, burst int) Limiter {
	return NewTokenBucketRate(rate.Limit(qps), burst)
}

func NewTokenBucketRate(qps rate.Limit, burst int) Limiter {
	return &TokenBucket{
		inner: rate.NewLimiter(qps, burst),
	}
}

func (bucket *TokenBucket) Limit() error {
	return bucket.LimitAt(time.Now(), 1)
}

func (bucket *TokenBucket) LimitN(n int) error {
	return bucket.LimitAt(time.Now(), n)
}

func (bucket *TokenBucket) LimitAt(now time.Time, n int) error {
	rsv := bucket.inner.ReserveN(now, n)
	if !rsv.OK() {
		return errMaxExceeded(bucket.inner.Burst())
	}

	if waitTime := rsv.Delay(); waitTime > 0 {
		rsv.Cancel()
		return errRateLimited(bucket.inner.Burst(), waitTime)
	}

	return nil
}
