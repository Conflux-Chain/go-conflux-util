package middlewares

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

var CtxKeyApiKey = CtxKey("X-API-Key")

type ApiKeyOption struct {
	Required  bool   // require a non-empty key; does not validate credentials
	ParamName string // case-sensitive, enabled if specified
	PathIndex int    // zero-based URL path index; ignored when ParamName is set
}

// NewApiKeyMiddleware creates middleware to inject client API key in context.
// Required rejects missing or empty keys with HTTP 400. Credential validation
// remains the caller's responsibility.
//
// API keys in URL paths or query parameters may appear in access logs and request
// traces. Callers are responsible for redacting these values.
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
// Path indices start at zero after trimming leading and trailing slashes;
// negative or out-of-range indices return "", false. The index is ignored when
// paramName is specified.
// The boolean result reports whether a non-empty key was extracted, not whether
// the key is a valid credential.
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

	if pathIndex < 0 || pathIndex >= len(fields) {
		return "", false
	}

	if unescaped, err := url.PathUnescape(fields[pathIndex]); err == nil {
		return unescaped, len(unescaped) > 0
	}

	return "", false
}
