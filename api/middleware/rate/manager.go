package rate

import (
	"context"
	"sync"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/rate"
	"github.com/mcuadros/go-defaults"
)

// LimiterManager manages the limiters for different tiers, keys, and APIs. It is safe for concurrent use.
type LimiterManager struct {
	config Config

	// tier -> api -> key -> limiter
	limiters map[string]map[string]map[string]rate.Limiter

	mu sync.Mutex
}

// NewLimiterManager creates a new LimiterManager with the given config.
func NewLimiterManager(config Config) *LimiterManager {
	defaults.SetDefaults(&config)

	limiters := make(map[string]map[string]map[string]rate.Limiter)

	for tier, api2Config := range config.Limiter {
		limiters[tier] = make(map[string]map[string]rate.Limiter)

		for api := range api2Config {
			limiters[tier][api] = make(map[string]rate.Limiter)
		}
	}

	return &LimiterManager{
		config:   config,
		limiters: limiters,
	}
}

// Limit limits the number of requests for the given tier, key, and api. It returns an error if the limit is exceeded.
func (manager *LimiterManager) Limit(tier, key, api string) error {
	return manager.LimitN(tier, key, api, 1)
}

// LimitN limits the number of requests for the given tier, key, and api. It returns an error if the limit is exceeded.
func (manager *LimiterManager) LimitN(tier, key, api string, n int) error {
	return manager.LimitAt(tier, key, api, time.Now(), n)
}

// LimitAt limits the number of requests for the given tier, key, and api at the given time. It returns an error if the limit is exceeded.
func (manager *LimiterManager) LimitAt(tier, key, api string, now time.Time, n int) error {
	if limiter, ok := manager.getOrCreateLimiter(tier, key, api); ok {
		return limiter.LimitAt(now, n)
	}

	return nil
}

// getOrCreateLimiter returns the limiter for the given tier, key, and api. If the limiter does not exist, it creates a new one.
// It returns the limiter and a boolean indicating whether the limiter was found or created.
func (manager *LimiterManager) getOrCreateLimiter(tier, key, api string) (rate.Limiter, bool) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	// limiters by tier
	limitersByTier, ok := manager.limiters[tier]
	if !ok {
		return nil, false
	}

	// limiters by api
	limitersByAPI, ok := limitersByTier[api]
	if !ok {
		return nil, false
	}

	// limiter by key
	if limiter, ok := limitersByAPI[key]; ok {
		return limiter, true
	}

	// API config must exists because we have already initialized the limiters map in constructor
	config := manager.config.Limiter[tier][api]

	limiter := rate.NewTokenBucketWithTimeout(config.Rate, config.Burst, manager.config.TTLsecs)
	limitersByAPI[key] = limiter

	return limiter, true
}

// Expire expires the limiters that have expired. It should be called periodically.
func (manager *LimiterManager) Expire() {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	for _, limitersByAPI := range manager.limiters {
		for _, limitersByKey := range limitersByAPI {
			for key, limiter := range limitersByKey {
				if limiter.Expired() {
					delete(limitersByKey, key)
				}
			}
		}
	}
}

// ScheduleExpire expires the limiters periodically. Generally, it should be called in a goroutine.
func (manager *LimiterManager) ScheduleExpire() {
	ticker := time.NewTicker(manager.config.ExpireInterval)
	defer ticker.Stop()

	for range ticker.C {
		manager.Expire()
	}
}

// ScheduleExpireCtx expires the limiters periodically until the context is done. It should be called in a goroutine.
// Note, the wg must be not nil, and the caller should call wg.Add(1) before calling this function, and call wg.Wait() after the context is done.
func (manager *LimiterManager) ScheduleExpireCtx(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(manager.config.ExpireInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			manager.Expire()
		}
	}
}
