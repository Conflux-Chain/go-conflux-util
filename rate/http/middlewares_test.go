package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/api"
	"github.com/Conflux-Chain/go-conflux-util/rate"
	"github.com/gin-gonic/gin"
	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddlewareErrors(t *testing.T) {
	now := time.Now().Add(time.Hour)
	fixed := rate.NewFixedWindow(time.Hour, 1)
	token := rate.NewTokenBucket(1, 1)
	fixedMax := fixed.LimitAt(now, 2)
	tokenMax := token.LimitAt(now, 2)
	require.NoError(t, fixed.LimitAt(now, 1))
	require.NoError(t, token.LimitAt(now, 1))
	fixedLimited := fixed.LimitAt(now, 1)
	tokenLimited := token.LimitAt(now, 1)
	for _, err := range []error{fixedMax, tokenMax, fixedLimited, tokenLimited} {
		require.Error(t, err)
	}

	for _, adapter := range []string{"http", "gin"} {
		t.Run(adapter, func(t *testing.T) {
			for _, tc := range []struct {
				name   string
				err    error
				status int
			}{
				{"success", nil, http.StatusNoContent},
				{"sentinel", rate.ErrRateLimited, http.StatusTooManyRequests},
				{"custom wrapped sentinel", fmt.Errorf("custom quota exceeded: %w", rate.ErrRateLimited), http.StatusTooManyRequests},
				{"fixed maximum", fixedMax, http.StatusTooManyRequests},
				{"fixed exhausted", fixedLimited, http.StatusTooManyRequests},
				{"token maximum", tokenMax, http.StatusTooManyRequests},
				{"token exhausted", tokenLimited, http.StatusTooManyRequests},
				{"standard wrapped", fmt.Errorf("request rejected: %w", fixedLimited), http.StatusTooManyRequests},
				{"package wrapped", pkgerrors.WithMessage(tokenLimited, "request rejected"), http.StatusTooManyRequests},
				{"internal", errors.New("configuration backend unavailable: internal-host:5432"), http.StatusInternalServerError},
				{"matching text without marker", errors.New(fixedLimited.Error()), http.StatusInternalServerError},
				{"invalid amount", fixed.LimitAt(now, 0), http.StatusInternalServerError},
			} {
				t.Run(tc.name, func(t *testing.T) {
					called := false
					limitCalled := false
					request := httptest.NewRequest(http.MethodGet, "/", nil)
					limit := func(ctx context.Context, resource string) error {
						limitCalled = true
						assert.Equal(t, request.Context(), ctx)
						assert.Equal(t, "resource", resource)
						return tc.err
					}
					var handler http.Handler
					if adapter == "http" {
						handler = NewHttpMiddleware(limit, "resource")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
							called = true
							w.WriteHeader(http.StatusNoContent)
						}))
					} else {
						engine := gin.New()
						engine.GET("/", NewApiMiddleware(limit, "resource"), func(ctx *gin.Context) {
							called = true
							ctx.Status(http.StatusNoContent)
						})
						handler = engine
					}
					recorder := httptest.NewRecorder()
					handler.ServeHTTP(recorder, request)
					assert.True(t, limitCalled)
					assert.Equal(t, tc.status, recorder.Code)
					assert.Equal(t, tc.err == nil, called)
					if tc.err == nil {
						assert.Empty(t, recorder.Body.String())
						return
					}
					if adapter == "http" {
						message := tc.err.Error()
						if tc.status == http.StatusInternalServerError {
							message = http.StatusText(http.StatusInternalServerError)
						}
						assert.Equal(t, message+"\n", recorder.Body.String())
					} else {
						var body api.BusinessError
						require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
						assert.Equal(t, tc.err.Error(), body.Data)
						if tc.status == http.StatusInternalServerError {
							assert.Equal(t, api.ErrCodeInternal, body.Code)
							assert.Equal(t, "Internal server error", body.Message)
						} else {
							assert.Equal(t, api.ErrCodeTooManyRequests, body.Code)
							assert.Equal(t, "Too many requests", body.Message)
						}
					}
				})
			}
		})
	}
}
