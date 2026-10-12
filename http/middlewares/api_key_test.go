package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetApiKey(t *testing.T) {
	for _, tc := range []struct {
		name      string
		target    string
		paramName string
		pathIndex int
		wantKey   string
		wantOK    bool
	}{
		{"empty root", "/", "", 0, "", false},
		{"empty segment", "/v3//method", "", 1, "", false},
		{"path key", "/v3/secret/method", "", 1, "secret", true},
		{"escaped path key", "/v3/a%2Fb/method", "", 1, "a/b", true},
		{"empty query", "/v3/secret?apikey=", "apikey", 1, "", false},
		{"query key", "/v3/secret?apikey=configured", "apikey", 1, "configured", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tc.target, nil)
			key, ok := GetApiKey(r, tc.paramName, tc.pathIndex)
			assert.Equal(t, tc.wantKey, key)
			assert.Equal(t, tc.wantOK, ok)
		})
	}
}

func TestApiKeyMiddlewareRequired(t *testing.T) {
	for _, tc := range []struct {
		name      string
		target    string
		pathIndex int
		required  bool
		wantKey   string
		wantCode  int
	}{
		{"required empty root", "/", 0, true, "", http.StatusBadRequest},
		{"required empty segment", "/v3//method", 1, true, "", http.StatusBadRequest},
		{"optional empty segment", "/v3//method", 1, false, "", http.StatusNoContent},
		{"required path key", "/v3/secret/method", 1, true, "secret", http.StatusNoContent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				key, ok := GetApiKeyFromContext(r.Context())
				assert.Equal(t, tc.wantKey, key)
				assert.Equal(t, tc.wantKey != "", ok)
				w.WriteHeader(http.StatusNoContent)
			})
			handler := NewApiKeyMiddleware(ApiKeyOption{
				Required:  tc.required,
				PathIndex: tc.pathIndex,
			})(next)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tc.target, nil))
			assert.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantCode == http.StatusNoContent, called)
		})
	}
}
