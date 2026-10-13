package rate

import (
	"errors"
	"fmt"
	"testing"
	"time"

	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestRateLimitErrors(t *testing.T) {
	for _, tc := range []struct {
		name    string
		err     error
		message string
		limited bool
	}{
		{"maximum", errMaxExceeded(5), "Too many requests, exceeds 5 at a time", true},
		{"quota", errRateLimited(5, time.Second), "Too many requests (exceeds 5), try again after 1s", true},
		{"invalid amount", errInvalidAmount(0), "Invalid request amount 0: must be positive", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.EqualError(t, tc.err, tc.message)
			assert.Equal(t, tc.limited, errors.Is(tc.err, ErrRateLimited))
			assert.Equal(t, tc.limited, errors.Is(fmt.Errorf("wrapped: %w", tc.err), ErrRateLimited))
			assert.Equal(t, tc.limited, errors.Is(pkgerrors.WithMessage(tc.err, "wrapped"), ErrRateLimited))
			assert.False(t, errors.Is(tc.err, errors.New("unrelated")))
		})
	}
}
