package rate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFixedWindow(t *testing.T) {
	limiter := NewFixedWindow(time.Minute, 10)

	// first 10 tokens allowed
	assert.NoError(t, limiter.Limit())
	assert.NoError(t, limiter.LimitN(9))

	// the 11th token not allowed
	assert.Error(t, limiter.Limit())

	// tokens allowed one minute later
	assert.Error(t, limiter.LimitN(10))
	assert.NoError(t, limiter.LimitAt(time.Now().Add(time.Minute), 1))
}
