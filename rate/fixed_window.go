package rate

import (
	"sync/atomic"
	"time"
)

// FixedWindow limits rate in a fixed window.
type FixedWindow struct {
	interval time.Duration
	max      int64

	startTime int64 // in milliseconds
	count     int64
}

func NewFixedWindow(interval time.Duration, max int) Limiter {
	return &FixedWindow{
		interval:  interval,
		max:       int64(max),
		startTime: time.Now().Truncate(interval).UnixMilli(),
	}
}

func (window *FixedWindow) Limit() error {
	return window.LimitAt(time.Now(), 1)
}

func (window *FixedWindow) LimitN(n int) error {
	return window.LimitAt(time.Now(), n)
}

func (window *FixedWindow) LimitAt(now time.Time, n int) error {
	if int64(n) > window.max {
		return errMaxExceeded(int(window.max))
	}

	window.advance(now)

	if count := atomic.AddInt64(&window.count, int64(n)); count <= window.max {
		return nil
	}

	// rollback
	atomic.AddInt64(&window.count, int64(-n))

	startTime := atomic.LoadInt64(&window.startTime)
	nextTime := time.UnixMilli(startTime).Add(window.interval)
	waitTime := nextTime.Sub(now)

	return errRateLimited(int(window.max), waitTime)
}

func (window *FixedWindow) advance(now time.Time) {
	truncated := now.Truncate(window.interval).UnixMilli()

	if startTime := atomic.LoadInt64(&window.startTime); startTime < truncated {
		// reset
		atomic.StoreInt64(&window.startTime, truncated)
		atomic.StoreInt64(&window.count, 0)
	}
}

func (window *FixedWindow) Expired() bool {
	startTime := atomic.LoadInt64(&window.startTime)
	return time.Since(time.UnixMilli(startTime)) > window.interval
}
