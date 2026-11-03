package rate

import "time"

type Config struct {
	// tier -> api -> LimiterConfig
	Limiter map[string]map[string]LimiterConfig

	// TTLsecs is the expiration timeout in seconds since the last successful
	// request (or creation if unused). It must be at least Burst/Rate seconds
	// for every configured limiter, so eviction cannot restore quota early.
	TTLsecs int64 `default:"600"`

	// ExpireInterval is the interval at which the limiter will be checked for expiration
	ExpireInterval time.Duration `default:"1m"`
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
