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
	timeoutSecs := int64(float64(burst)/float64(qps)) + 1

	return &TokenBucket{
		inner:       rate.NewLimiter(qps, burst),
		lastSeen:    time.Now().Unix(),
		timeoutSecs: timeoutSecs,
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
		return errRateLimited(int(bucket.inner.Limit()), waitTime)
	}

	atomic.StoreInt64(&bucket.lastSeen, now.Unix())

	return nil
}

func (bucket *TokenBucket) Expired() bool {
	lastSeen := atomic.LoadInt64(&bucket.lastSeen)
	return time.Now().Unix() > lastSeen+bucket.timeoutSecs
}
