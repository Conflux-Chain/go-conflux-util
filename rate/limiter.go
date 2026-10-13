package rate

import (
	"time"
)

type Limiter interface {
	Limit() error
	// LimitN consumes n tokens at the current time. Non-positive amounts return
	// an error without changing the limiter's state.
	LimitN(n int) error
	// LimitAt consumes n tokens at now, with the same amount rules as LimitN.
	LimitAt(now time.Time, n int) error

	// Expired indicates whether limiter not updated for a long time.
	// Generally, it is used for garbage collection.
	Expired() bool
	ExpiredAt(now time.Time) bool
}
