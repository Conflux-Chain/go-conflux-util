package http

import (
	"context"
	"sync"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/rate"
	"github.com/pkg/errors"
)

type LimiterFactory interface {
	GetGroupAndKey(ctx VisitContext) (group, key string, err error)

	Create(ctx VisitContext) (rate.Limiter, error)
}

type Registry struct {
	factory LimiterFactory

	// resource => group => key => limiter, where group and key are as following:
	//
	// 1. client IP.
	// 2. Fluent group + client IP.
	// 3. VIP group (t0, t1, t2, ...) + api key
	//
	// Limiters in a group could be dynamically updated in batch.
	limiters map[string]map[string]map[string]rate.Limiter

	mu sync.Mutex
}

func NewRegistry(factory LimiterFactory) *Registry {
	return &Registry{
		factory:  factory,
		limiters: make(map[string]map[string]map[string]rate.Limiter),
	}
}

func (r *Registry) getLimiters(resource, group string) map[string]rate.Limiter {
	limitersByGroup, ok := r.limiters[resource]
	if !ok {
		limitersByGroup = make(map[string]map[string]rate.Limiter)
		r.limiters[resource] = limitersByGroup
	}

	limitersByKey, ok := limitersByGroup[group]
	if !ok {
		limitersByKey = make(map[string]rate.Limiter)
		limitersByGroup[group] = limitersByKey
	}

	return limitersByKey
}

// Limit limits request rate according to HTTP request context.
// Note, it requires to hook IP/API key middlewares for HTTP server.
func (r *Registry) Limit(ctx context.Context, resource string) error {
	visit := ParseVisitContext(ctx, resource)

	group, key, err := r.factory.GetGroupAndKey(visit)
	if err != nil {
		return errors.WithMessage(err, "Failed to get group and key from visit context")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	limiters := r.getLimiters(resource, group)

	if limiter, ok := limiters[key]; ok {
		return limiter.Limit()
	}

	limiter, err := r.factory.Create(visit)
	if err != nil {
		return errors.WithMessage(err, "Failed to create limiter")
	}

	limiters[key] = limiter

	return limiter.Limit()
}

func (r *Registry) GC() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, limitersByGroup := range r.limiters {
		for _, limitersByKey := range limitersByGroup {
			for key, limiter := range limitersByKey {
				if limiter.Expired() {
					delete(limitersByKey, key)
				}
			}
		}
	}
}

func (r *Registry) ScheduleGC(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		r.GC()
	}
}

func (r *Registry) RemoveAll() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.limiters = make(map[string]map[string]map[string]rate.Limiter)
}

func (r *Registry) Remove(resource, group string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if limiters, ok := r.limiters[resource]; ok {
		delete(limiters, group)
	}
}

func (r *Registry) RemoveBatch(items map[string]map[string]bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for resource, groups := range items {
		if limiters, ok := r.limiters[resource]; ok {
			for group := range groups {
				delete(limiters, group)
			}
		}
	}
}
