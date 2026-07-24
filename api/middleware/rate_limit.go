package middleware

import (
	"net/http"

	apiUtil "github.com/Conflux-Chain/go-conflux-util/api"
	"github.com/Conflux-Chain/go-conflux-util/api/middleware/rate"
	"github.com/gin-gonic/gin"
)

const RateLimitTierFree = "free"

// RateLimitKeyExtractorFunc defines how to extract the rate limit key from the gin context.
type RateLimitKeyExtractorFunc func(c *gin.Context) (tier, key string, err error)

// GetRealIP retrieves the real IP address of the client from the gin context, considering the "X-Real-IP" header if present.
//
// Note, Gin provides a method ClientIP() to get the client IP address, but it requires to set the trusted proxies. In this case,
// services have to change the proxy settings when proxy IP address changed.
//
// As a result, the service depends on the devops to set the correct "X-Real-IP" header in the reverse proxy.
func GetRealIP(c *gin.Context) string {
	ip := c.GetHeader("X-Real-IP")

	// no proxy, use the remote address directly
	if len(ip) == 0 {
		ip = c.RemoteIP()
	}

	return ip
}

// DefaultRateLimitKeyExtractor is the default implementation of RateLimitKeyExtractorFunc, which uses the client's real IP address as the rate limit key and assigns it to the "free" tier.
func DefaultRateLimitKeyExtractor(c *gin.Context) (tier, key string, err error) {
	return RateLimitTierFree, GetRealIP(c), nil
}

// RateLimitManager is a wrapper around rate.LimiterManager that provides a gin middleware for rate limit.
type RateLimitManager struct {
	*rate.LimiterManager

	keyExtractor RateLimitKeyExtractorFunc
}

// NewRateLimitManager creates a new RateLimitManager with the given configuration and key extractor function.
// If no key extractor function is provided, it defaults to using the DefaultRateLimitKeyExtractor.
func NewRateLimitManager(config rate.Config, keyExtractor ...RateLimitKeyExtractorFunc) *RateLimitManager {
	var extractor RateLimitKeyExtractorFunc = DefaultRateLimitKeyExtractor
	if len(keyExtractor) > 0 {
		extractor = keyExtractor[0]
	}

	return &RateLimitManager{
		LimiterManager: rate.NewLimiterManager(config),
		keyExtractor:   extractor,
	}
}

// Middleware returns a gin.HandlerFunc that applies rate limiting based on the provided API name.
func (manager *RateLimitManager) Middleware(api string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tier, key, err := manager.keyExtractor(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusOK, apiUtil.ErrValidation(err))
		} else if err = manager.Limit(tier, key, api); err != nil {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, apiUtil.ErrTooManyRequests(err))
		} else {
			c.Next()
		}
	}
}
