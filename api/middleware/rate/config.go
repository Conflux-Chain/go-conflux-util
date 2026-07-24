package rate

import "time"

type Config struct {
	Limiter LimiterConfig

	TTLsecs int64 `default:"600"` // TTLsecs is the time (in seconds) to live for the limiter, after which it will be removed from the cache

	ExpireInterval time.Duration `default:"1m"` // ExpireInterval is the interval at which the limiter will be checked for expiration
}

type LimiterConfig struct {
	Tiers map[string]TierConfig // Key is tier name, e.g. "free", "vip1", "vip2", "svip"
}

type TierConfig struct {
	APIs map[string]APIConfig // Key is API name, e.g. "getUser", "placeOrder", "getOrder"
}

type APIConfig struct {
	Rate  float64 // Rate is the number of requests per second allowed
	Burst int     // Burst is the maximum number of requests that can be made in a short period of time
}

// Add adds a new API rate limit configuration to the LimiterConfig. It returns true if the configuration was added successfully, or false if the configuration already exists.
func (config *LimiterConfig) Add(tier, api string, rate float64, burst int) bool {
	if _, ok := config.Tiers[tier]; !ok {
		config.Tiers[tier] = TierConfig{
			APIs: make(map[string]APIConfig),
		}
	}

	if _, ok := config.Tiers[tier].APIs[api]; ok {
		return false
	}

	config.Tiers[tier].APIs[api] = APIConfig{
		Rate:  rate,
		Burst: burst,
	}

	return true
}
