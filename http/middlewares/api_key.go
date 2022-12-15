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
			var (
				apiKey string
				ok     bool
			)

			if len(option.ParamName) == 0 {
				apiKey, ok = GetApiKeyFromPath(r, option.PathIndex)
			} else {
				apiKey, ok = GetApiKeyFromQuery(r, option.ParamName)
			}

			if ok {
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

// GetApiKeyFromQuery returns the client API key from query if specified.
// E.g. http://example.com/abc?apikey=${val}
func GetApiKeyFromQuery(r *http.Request, paramName string) (string, bool) {
	if r == nil || r.URL == nil {
		return "", false
	}

	if key := r.URL.Query().Get(paramName); len(key) > 0 {
		return key, true
	}

	return "", false
}

// GetApiKeyFromPath returns the client API key from URL path if specified.
// E.g. http://example.com/v3/${apikey}
func GetApiKeyFromPath(r *http.Request, index int) (string, bool) {
	if r == nil || r.URL == nil {
		return "", false
	}

	escaped := strings.Trim(r.URL.EscapedPath(), "/")
	fields := strings.Split(escaped, "/")

	if index >= len(fields) {
		return "", false
	}

	if unescaped, err := url.PathUnescape(fields[index]); err == nil {
		return unescaped, true
	}

	return "", false
}
