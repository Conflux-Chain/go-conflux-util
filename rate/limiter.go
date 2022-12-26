package rate

import (
	"time"
)

type Limiter interface {
	Limit() error
	LimitN(n int) error
	LimitAt(now time.Time, n int) error

	// Expired indicates whether limiter not updated for a long time.
	// Generally, it is used for garbage collection.
	Expired() bool
}
