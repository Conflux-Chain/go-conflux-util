package rate

import (
	"sync"
	"time"
)

// FixedWindow limits rate in a fixed window.
// It is safe for concurrent use and must not be copied after first use.
type FixedWindow struct {
	interval time.Duration
	max      int

	mu        sync.Mutex // protects startTime and count
	startTime time.Time
	count     int
}

// NewFixedWindow creates a limiter with at most max tokens per interval.
// It panics if interval or max is not positive.
func NewFixedWindow(interval time.Duration, max int) Limiter {
	if interval <= 0 {
		panic("rate: fixed window interval must be positive")
	}

	if max <= 0 {
		panic("rate: fixed window max must be positive")
	}

	return &FixedWindow{
		interval:  interval,
		max:       max,
		startTime: time.Now().Truncate(interval),
	}
}

func (window *FixedWindow) Limit() error {
	return window.LimitAt(time.Now(), 1)
}

// LimitN consumes n tokens at the current time, with the same amount rules as LimitAt.
func (window *FixedWindow) LimitN(n int) error {
	return window.LimitAt(time.Now(), n)
}

// LimitAt consumes n tokens at now. The amount must be positive;
// otherwise it returns an error without changing state.
func (window *FixedWindow) LimitAt(now time.Time, n int) error {
	if n <= 0 {
		return errInvalidAmount(n)
	}
	if n > window.max {
		return errMaxExceeded(window.max)
	}

	window.mu.Lock()
	defer window.mu.Unlock()

	window.advance(now)

	// Compare the remaining quota before adding to avoid integer overflow.
	if n <= window.max-window.count {
		window.count += n
		return nil
	}

	nextTime := window.startTime.Add(window.interval)
	waitTime := nextTime.Sub(now)

	return errRateLimited(window.max, waitTime)
}

// advance starts a new window when needed. The caller must hold window.mu.
func (window *FixedWindow) advance(now time.Time) {
	truncated := now.Truncate(window.interval)

	if window.startTime.Before(truncated) {
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

	return window.startTime.Add(window.interval).Before(now)
}
