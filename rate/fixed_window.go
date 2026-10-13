package rate

import (
	"sync"
	"time"
)

// FixedWindow limits rate in a fixed window.
// It is safe for concurrent use and must not be copied after first use.
type FixedWindow struct {
	interval time.Duration
	max      int64

	mu        sync.Mutex // protects startTime and count
	startTime int64      // in milliseconds
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

	window.mu.Lock()
	defer window.mu.Unlock()

	window.advance(now)

	if count := window.count + int64(n); count <= window.max {
		window.count = count
		return nil
	}

	nextTime := time.UnixMilli(window.startTime).Add(window.interval)
	waitTime := nextTime.Sub(now)

	return errRateLimited(int(window.max), waitTime)
}

// advance starts a new window when needed. The caller must hold window.mu.
func (window *FixedWindow) advance(now time.Time) {
	truncated := now.Truncate(window.interval).UnixMilli()

	if window.startTime < truncated {
		// reset
		window.startTime = truncated
		window.count = 0
	}
}

func (window *FixedWindow) Expired() bool {
	return window.ExpiredAt(time.Now())
}

func (window *FixedWindow) ExpiredAt(now time.Time) bool {
	window.mu.Lock()
	defer window.mu.Unlock()

	return time.UnixMilli(window.startTime).Add(window.interval).Before(now)
}
