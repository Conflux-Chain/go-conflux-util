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

// ChainedLimiter rate limiters with resposibility chain
type ChainedLimiter struct {
	limiters []Limiter
}

func NewChainedLimiter(limiters ...Limiter) *ChainedLimiter {
	return &ChainedLimiter{limiters: limiters}
}

// implements `rate.Limiter` interface

func (cl *ChainedLimiter) Limit() error {
	return cl.LimitN(1)
}

func (cl *ChainedLimiter) LimitN(n int) error {
	return cl.LimitAt(time.Now(), n)
}

func (cl *ChainedLimiter) LimitAt(now time.Time, n int) error {
	for i := range cl.limiters {
		if err := cl.limiters[i].LimitAt(now, n); err != nil {
			return err
		}
	}

	return nil
}

func (cl *ChainedLimiter) Expired() bool {
	for i := range cl.limiters {
		if cl.limiters[i].Expired() {
			return true
		}
	}

	return false
}
