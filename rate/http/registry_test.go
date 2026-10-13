package http

import (
	"context"
	"testing"

	"github.com/Conflux-Chain/go-conflux-util/rate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type registryTestLimiter struct {
	rate.Limiter
	expired bool
}

func (l *registryTestLimiter) Expired() bool { return l.expired }

type registryTestFactory struct {
	group string
}

func (f *registryTestFactory) GetGroupAndKey(context.Context, string) (string, string, error) {
	return f.group, "key", nil
}

func (*registryTestFactory) Create(context.Context, string, string) (rate.Limiter, error) {
	return &registryTestLimiter{Limiter: rate.NewTokenBucket(1, 1)}, nil
}

func TestRegistryGC(t *testing.T) {
	r := NewRegistry(&registryTestFactory{group: "group"})
	expired := &registryTestLimiter{expired: true}
	active := &registryTestLimiter{}
	r.limiters = map[string]map[string]map[string]rate.Limiter{
		"expired": {"group": {"key": expired}},
		"mixed": {
			"expired": {"key": expired},
			"active":  {"expired": expired, "active": active},
			"empty":   {},
		},
		"empty": {},
	}

	r.GC()
	assert.Equal(t, map[string]map[string]map[string]rate.Limiter{
		"mixed": {"active": {"active": active}},
	}, r.limiters)

	active.expired = true
	r.GC()
	assert.Empty(t, r.limiters)
	r.GC() // Repeated cleanup of an empty registry is safe.

	// A resource removed by GC can acquire a new limiter normally.
	require.NoError(t, r.Limit(context.Background(), "expired"))
	assert.NotSame(t, expired, r.limiters["expired"]["group"]["key"])
}

func TestRegistryRemove(t *testing.T) {
	for _, tc := range []struct {
		name   string
		remove func(*Registry, string, string)
	}{
		{"single", (*Registry).Remove},
		{"batch", func(r *Registry, resource, group string) {
			r.RemoveBatch(map[string]map[string]bool{
				resource:  {group: true, "missing": true},
				"missing": {"missing": true},
			})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			factory := &registryTestFactory{group: "first"}
			r := NewRegistry(factory)
			ctx := context.Background()
			require.NoError(t, r.Limit(ctx, "resource"))
			original := r.limiters["resource"]["first"]["key"]
			factory.group = "second"
			require.NoError(t, r.Limit(ctx, "resource"))
			require.NoError(t, r.Limit(ctx, "other"))
			remaining := r.limiters["resource"]["second"]["key"]
			other := r.limiters["other"]["second"]["key"]

			tc.remove(r, "resource", "first")
			assert.NotContains(t, r.limiters["resource"], "first")
			assert.Same(t, remaining, r.limiters["resource"]["second"]["key"])

			tc.remove(r, "resource", "second")
			assert.NotContains(t, r.limiters, "resource")
			assert.Same(t, other, r.limiters["other"]["second"]["key"])
			tc.remove(r, "resource", "second")

			factory.group = "first"
			require.NoError(t, r.Limit(ctx, "resource"))
			assert.NotSame(t, original, r.limiters["resource"]["first"]["key"])
		})
	}
}
