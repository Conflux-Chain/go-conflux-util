package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/Conflux-Chain/go-conflux-util/api"
	"github.com/Conflux-Chain/go-conflux-util/http/middlewares"
	"github.com/Conflux-Chain/go-conflux-util/rate"
	"github.com/gin-gonic/gin"
)

// LimitFunc returns nil to allow a request. Middleware returns HTTP 429 for
// errors matching rate.ErrRateLimited via errors.Is, exposing their messages to
// clients. Custom limiters should return or wrap that sentinel for quota errors.
// All other errors produce HTTP 500. NewHttpMiddleware returns a generic
// message; NewApiMiddleware uses api.ErrInternal, including err.Error() in data.
type LimitFunc func(ctx context.Context, resource string) error

// NewHttpMiddleware creates middleware to limit HTTP request rate.
// Errors are handled according to the LimitFunc contract.
func NewHttpMiddleware(f LimitFunc, resource string) middlewares.Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := f(r.Context(), resource); err != nil {
				if errors.Is(err, rate.ErrRateLimited) {
					http.Error(w, err.Error(), http.StatusTooManyRequests)
				} else {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			} else {
				h.ServeHTTP(w, r)
			}
		})
	}
}

// NewApiMiddleware creates middleware to limit request rate for REST API.
// Errors are handled according to the LimitFunc contract.
func NewApiMiddleware(f LimitFunc, resource string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if err := f(ctx.Request.Context(), resource); err != nil {
			if errors.Is(err, rate.ErrRateLimited) {
				ctx.AbortWithStatusJSON(http.StatusTooManyRequests, api.ErrTooManyRequests(err))
			} else {
				ctx.AbortWithStatusJSON(http.StatusInternalServerError, api.ErrInternal(err))
			}
		} else {
			ctx.Next()
		}
	}
}
