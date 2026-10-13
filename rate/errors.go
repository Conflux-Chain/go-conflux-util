package rate

import (
	"time"

	"github.com/pkg/errors"
)

func errInvalidAmount(n int) error {
	return errors.Errorf("Invalid request amount %v: must be positive", n)
}

func errMaxExceeded(max int) error {
	return errors.Errorf("Too many requests, exceeds %v at a time", max)
}

func errRateLimited(max int, waitTime time.Duration) error {
	return errors.Errorf(
		"Too many requests (exceeds %v), try again after %v", max, waitTime.Round(time.Millisecond),
	)
}
