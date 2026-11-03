package rate

import (
	"math"
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

func TestLimiterManagerNewInvalidConfig(t *testing.T) {
	for _, tc := range []struct {
		name    string
		rate    float64
		burst   int
		ttlSecs int64
	}{
		{"zero rate", 0, 1, 60},
		{"negative rate", -1, 1, 60},
		{"NaN rate", math.NaN(), 1, 60},
		{"infinite rate", math.Inf(1), 1, 60},
		{"zero burst", 1, 0, 60},
		{"negative burst", 1, -1, 60},
		{"negative TTL", 1, 1, -1},
		{"TTL overflow", 1, 1, math.MaxInt64},
		{"refill duration overflow", 1e-10, 1, 60},
		{"TTL shorter than refill time", 1, 10, 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := Config{TTLsecs: tc.ttlSecs}
			config.Add("free", "/v1/status", 5, 20)
			config.Add("vip", "/v2/status", tc.rate, tc.burst)

			// Invalid settings must fail during construction, before any request.
			assert.Panics(t, func() { NewLimiterManager(config) })
		})
	}
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

func TestLimiterManagerExpireAfterRefill(t *testing.T) {
	config := Config{TTLsecs: 10}
	config.Add("free", "/api", 1, 10)
	manager := NewLimiterManager(config)
	now := time.Now()
	assert.NoError(t, manager.LimitAt("free", "key", "/api", now, 10))

	// Before refill completes, cleanup must retain the bucket and its consumed quota.
	before := now.Add(9 * time.Second)
	manager.ExpireAt(before)
	assert.Error(t, manager.LimitAt("free", "key", "/api", before, 10))

	deadline := now.Add(10 * time.Second)
	manager.ExpireAt(deadline)
	assert.Len(t, manager.limiters["free"]["/api"], 1)

	// Once expired, the old bucket would be full; recreating it adds no extra quota.
	later := deadline.Add(time.Nanosecond)
	manager.ExpireAt(later)
	assert.Empty(t, manager.limiters["free"]["/api"])
	assert.NoError(t, manager.LimitAt("free", "key", "/api", later, 10))
	assert.Error(t, manager.LimitAt("free", "key", "/api", later, 1))
}
