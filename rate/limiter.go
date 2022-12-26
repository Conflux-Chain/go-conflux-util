package rate

import (
	"time"
)

type Limiter interface {
	Limit() error
	LimitN(n int) error
	LimitAt(now time.Time, n int) error
}
