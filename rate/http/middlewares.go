package http

import (
	"context"
	"net/http"

	"github.com/Conflux-Chain/go-conflux-util/api"
	"github.com/Conflux-Chain/go-conflux-util/http/middlewares"
	"github.com/gin-gonic/gin"
)

type LimitFunc func(ctx context.Context, resource string) error

// NewHttpMiddleware creates middleware to limit HTTP request rate.
func NewHttpMiddleware(f LimitFunc, resource string) middlewares.Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := f(r.Context(), resource); err != nil {
				http.Error(w, err.Error(), http.StatusTooManyRequests)
			} else {
				h.ServeHTTP(w, r)
			}
		})
	}
}

// NewApiMiddleware creates middleware to limit request rate for REST API.
func NewApiMiddleware(f LimitFunc, resource string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if err := f(ctx.Request.Context(), resource); err != nil {
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, api.ErrTooManyRequests(err))
		} else {
			ctx.Next()
		}
	}
}
