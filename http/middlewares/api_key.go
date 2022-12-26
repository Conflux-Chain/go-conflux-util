package middlewares

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

var CtxKeyApiKey = CtxKey("X-API-Key")

type ApiKeyOption struct {
	Required  bool
	ParamName string // case-sensitive, enabled if specified
	PathIndex int    // api key in URL path
}

// NewApiKeyMiddleware creates middleware to inject client API key in context.
func NewApiKeyMiddleware(option ApiKeyOption) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if apiKey, ok := GetApiKey(r, option.ParamName, option.PathIndex); ok {
				ctx := context.WithValue(r.Context(), CtxKeyApiKey, apiKey)
				h.ServeHTTP(w, r.WithContext(ctx))
			} else if option.Required {
				http.Error(w, "API key not specified", http.StatusBadRequest)
			} else {
				h.ServeHTTP(w, r)
			}
		})
	}
}

// GetApiKeyFromContext get client API key from the specified `ctx`.
func GetApiKeyFromContext(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(CtxKeyApiKey).(string)
	return val, ok
}

// GetApiKey returns the client API key from the specified `url`. If `paramName`specified,
// parse from query. Otherwise, parse from URL path with specified `pathIndex`. Note, the
// `paramName` is case sensitive.
func GetApiKey(r *http.Request, paramName string, pathIndex int) (string, bool) {
	if r == nil || r.URL == nil {
		return "", false
	}

	// parse from query, e.g. http://example.com/abc?apikey=${val}
	if len(paramName) > 0 {
		key := r.URL.Query().Get(paramName)
		return key, len(key) > 0
	}

	// otherwise, parse from URL path, e.g. http://example.com/v3/${apikey}
	escaped := strings.Trim(r.URL.EscapedPath(), "/")
	fields := strings.Split(escaped, "/")

	if pathIndex >= len(fields) {
		return "", false
	}

	if unescaped, err := url.PathUnescape(fields[pathIndex]); err == nil {
		return unescaped, true
	}

	return "", false
}
