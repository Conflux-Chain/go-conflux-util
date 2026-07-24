package rate

import "time"

type Config struct {
	Limiter map[string]map[string]LimiterConfig // tier -> api -> LimiterConfig

	TTLsecs int64 `default:"600"` // TTLsecs is the time (in seconds) to live for the limiter, after which it will be removed from the cache

	ExpireInterval time.Duration `default:"1m"` // ExpireInterval is the interval at which the limiter will be checked for expiration
}

type LimiterConfig struct {
	Rate  float64 // Rate is the number of requests per second allowed
	Burst int     // Burst is the maximum number of requests that can be made in a short period of time
}

// Add adds a new limiter configuration for a given tier and API. If the configuration already exists, it returns false. Otherwise, it adds the configuration and returns true.
func (config *Config) Add(tier, api string, rate float64, burst int) bool {
	if config.Limiter == nil {
		config.Limiter = make(map[string]map[string]LimiterConfig)
	}

	if _, ok := config.Limiter[tier]; !ok {
		config.Limiter[tier] = make(map[string]LimiterConfig)
	}

	if _, ok := config.Limiter[tier][api]; ok {
		return false
	}

	config.Limiter[tier][api] = LimiterConfig{
		Rate:  rate,
		Burst: burst,
	}

	return true
}
