package rate

import (
	"time"

	"github.com/pkg/errors"
)

// ErrRateLimited identifies requests rejected for exceeding the available quota
// or the maximum amount per request. Use errors.Is to check wrapped errors.
var ErrRateLimited = errors.New("rate limit exceeded")

// rateLimitError preserves the detailed message while identifying quota errors.
type rateLimitError struct {
	error
}

func (rateLimitError) Is(target error) bool {
	return target == ErrRateLimited
}

func errInvalidAmount(n int) error {
	return errors.Errorf("Invalid request amount %v: must be positive", n)
}

func errMaxExceeded(max int) error {
	return rateLimitError{errors.Errorf("Too many requests, exceeds %v at a time", max)}
}

func errRateLimited(max int, waitTime time.Duration) error {
	return rateLimitError{errors.Errorf(
		"Too many requests (exceeds %v), try again after %v", max, waitTime.Round(time.Millisecond),
	)}
}
