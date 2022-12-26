package http

import (
	"context"

	"github.com/Conflux-Chain/go-conflux-util/http/middlewares"
)

type VisitContext struct {
	Resource string // e.g. http requests, RPC requests
	IP       string // real remote adddress
	ApiKey   string // optional api key
}

// ParseVisitContext parses visit context from HTTP request context.
func ParseVisitContext(ctx context.Context, resource string) VisitContext {
	visit := VisitContext{
		Resource: resource,
	}

	if ip, ok := middlewares.GetRealIPFromContext(ctx); ok {
		visit.IP = ip
	}

	if apiKey, ok := middlewares.GetApiKeyFromContext(ctx); ok {
		visit.ApiKey = apiKey
	}

	return visit
}
