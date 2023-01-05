package http

import (
	"context"
	"testing"

	"github.com/Conflux-Chain/go-conflux-util/http/middlewares"
	"github.com/stretchr/testify/assert"
)

const (
	TestResource = "rpc-all"
	TestRealIp   = "127.0.0.1"
	TestApiKey   = "abcde123456"
	TestDummyKey = "dummy"

	NewTestResource = "another-rpc-all"
	NewTestRealIp   = "192.168.0.1"
	NewTestApiKey   = "ABCDE123456"
)

func TestVisitContext(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, middlewares.CtxKeyRealIP, TestRealIp)
	ctx = context.WithValue(ctx, middlewares.CtxKeyApiKey, TestApiKey)

	vc := NewVisitContext(ctx, TestResource)

	resrc, ok := vc.Resource()
	assert.True(t, ok)
	assert.Equal(t, resrc, TestResource)

	ip, ok := vc.Ip()
	assert.True(t, ok)
	assert.Equal(t, ip, TestRealIp)

	apiKey, ok := vc.ApiKey()
	assert.True(t, ok)
	assert.Equal(t, apiKey, TestApiKey)

	dummyV, ok := vc.Value(TestDummyKey).(string)
	assert.False(t, ok)
	assert.Empty(t, dummyV)

	vc = vc.WithResource(NewTestResource)
	resrc, _ = vc.Resource()
	assert.Equal(t, resrc, NewTestResource)

	vc = vc.WithIp(NewTestRealIp)
	resrc, _ = vc.Ip()
	assert.Equal(t, resrc, NewTestRealIp)

	vc = vc.WithApiKey(NewTestApiKey)
	resrc, _ = vc.ApiKey()
	assert.Equal(t, resrc, NewTestApiKey)
}
