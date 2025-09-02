package middleware

import (
	"time"

	"github.com/Conflux-Chain/go-conflux-util/metrics"
	"github.com/gin-gonic/gin"
)

func Wrap(controller func(c *gin.Context) (any, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := controller(c)
		if err != nil {
			ResponseError(c, err)
		} else {
			ResponseSuccess(c, result)
		}
	}
}

func Metrics(name string, args ...any) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		metrics.GetOrRegisterTimer(name, args...).UpdateSince(start)
	}
}
