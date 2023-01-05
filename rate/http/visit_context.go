package http

import (
	"context"

	"github.com/Conflux-Chain/go-conflux-util/http/middlewares"
)

const (
	CtxKeyVisitResource middlewares.CtxKey = "X-Visit-Resource"
)

// VisitContext encapsulates remote info such as IP, API key etc., from HTTP request context,
// or you can wrap any custom field within the context chain.
type VisitContext struct {
	ctx context.Context
}

func NewVisitContext(ctx context.Context, resource string) VisitContext {
	return VisitContext{
		ctx: context.WithValue(ctx, CtxKeyVisitResource, resource),
	}
}

func (vc VisitContext) Context() context.Context {
	return vc.ctx
}

func (vc VisitContext) WithValue(k, v interface{}) VisitContext {
	return VisitContext{
		ctx: context.WithValue(vc.ctx, k, v),
	}
}

func (vc VisitContext) Value(k interface{}) interface{} {
	return vc.ctx.Value(k)
}

func (vc VisitContext) Resource() (string, bool) {
	resource, ok := vc.Value(CtxKeyVisitResource).(string)
	return resource, ok
}

func (vc VisitContext) WithResource(resource string) VisitContext {
	return vc.WithValue(CtxKeyVisitResource, resource)
}

func (vc VisitContext) Ip() (string, bool) {
	resource, ok := vc.Value(middlewares.CtxKeyRealIP).(string)
	return resource, ok
}

func (vc VisitContext) WithIp(resource string) VisitContext {
	return vc.WithValue(middlewares.CtxKeyRealIP, resource)
}

func (vc VisitContext) ApiKey() (string, bool) {
	resource, ok := vc.Value(middlewares.CtxKeyApiKey).(string)
	return resource, ok
}

func (vc VisitContext) WithApiKey(resource string) VisitContext {
	return vc.WithValue(middlewares.CtxKeyApiKey, resource)
}
