package rate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLimiterManagerNew(t *testing.T) {
	var config Config
	config.Add("free", "/v1/status", 5, 20)

	manager := NewLimiterManager(config)

	// default values
	assert.Equal(t, int64(600), manager.config.TTLsecs)
	assert.Equal(t, time.Minute, manager.config.ExpireInterval)
}

func TestLimiterManagerLimit(t *testing.T) {
	var config Config
	config.Add("free", "/v1/status", 1, 5)

	manager := NewLimiterManager(config)

	// exceeds the burst at a time
	assert.Error(t, manager.LimitN("free", "ip1", "/v1/status", 6))

	// under the burst
	assert.NoError(t, manager.Limit("free", "ip1", "/v1/status"))
	assert.NoError(t, manager.LimitN("free", "ip1", "/v1/status", 4))

	// exceeds the burst
	assert.Error(t, manager.Limit("free", "ip1", "/v1/status"))

	// tier or api not configured
	assert.NoError(t, manager.LimitN("vip", "ip1", "/v1/status", 6))
	assert.NoError(t, manager.LimitN("free", "ip1", "/v2/status", 6))
}

func TestLimiterManagerExpire(t *testing.T) {
	var config Config
	config.TTLsecs = 10 // expired in 10 seconds
	config.Add("free", "/v1/status", 1, 5)

	manager := NewLimiterManager(config)

	assert.NoError(t, manager.LimitN("free", "ip1", "/v1/status", 5))
	assert.Error(t, manager.LimitN("free", "ip1", "/v1/status", 5))

	manager.ExpireAt(time.Now().Add(20 * time.Second))

	assert.NoError(t, manager.LimitN("free", "ip1", "/v1/status", 5))
}
