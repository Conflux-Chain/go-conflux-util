package middlewares

import "net/http"

// Middleware decorates HTTP Handler to response HTTP request.
type Middleware func(http.Handler) http.Handler

// CtxKey is the key type for injected kv pair in the context of HTTP request.
type CtxKey string

// Hook hooks middlewares to the specified HTTP handler. Middlewares will be decorated
// in reversed order, that is, the first middleware is the outermost one, and the last
// middleware is the innermost.
func Hook(handler http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}

	return handler
}
