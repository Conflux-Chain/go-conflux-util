package rate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	var config Config
	assert.True(t, config.Add("free", "/v1/status", 1, 10))
	assert.False(t, config.Add("free", "/v1/status", 5, 10))

	assert.True(t, config.Add("vip", "/v1/status", 1, 10))
	assert.True(t, config.Add("free", "/v2/status", 1, 10))

	assert.Equal(t, LimiterConfig{Rate: 1, Burst: 10}, config.Limiter["free"]["/v1/status"])
}
