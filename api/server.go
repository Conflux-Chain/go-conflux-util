package api

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/http/middlewares"
	"github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Endpoint string `default:":12345"`

	// ReadHeaderTimeout limits the time spent reading request headers.
	ReadHeaderTimeout time.Duration `default:"5s"`
	// ReadTimeout limits the time spent reading the entire request, including its body.
	ReadTimeout time.Duration `default:"15s"`
	// WriteTimeout limits response writing, including time spent handling the request.
	WriteTimeout time.Duration `default:"30s"`
	// IdleTimeout limits the wait for the next request on a keep-alive connection.
	IdleTimeout time.Duration `default:"60s"`

	RecoveryDisabled bool
	CorsOrigins      []string
	Logger           bool
}

type RouteFactory func(router *gin.Engine)

// MustServeFromViper loads the "api" configuration and calls MustServe,
// sharing its blocking behavior and process-lifetime contract.
// Configuration loading failures are logged with Fatal and exit the process.
func MustServeFromViper(factory RouteFactory, middlewares ...middlewares.Middleware) {
	var config Config
	viper.MustUnmarshalKey("api", &config)
	MustServe(config, factory, middlewares...)
}

// MustServe starts an HTTP server and blocks while serving requests.
// It is intended to run for the lifetime of the process and provides no graceful
// shutdown entry point. ListenAndServe errors other than http.ErrServerClosed
// are logged with Fatal and exit the process.
func MustServe(config Config, factory RouteFactory, mws ...middlewares.Middleware) {
	// gin init mode via init(), but .env may be loaded later and ignored the GIN_MODE.
	if os.Getenv(gin.EnvGinMode) == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	if !config.RecoveryDisabled {
		router.Use(gin.Recovery())
	}

	router.Use(newCorsMiddleware(config.CorsOrigins))

	if config.Logger {
		router.Use(gin.Logger())
	}

	factory(router)

	server := http.Server{
		Addr:              config.Endpoint,
		Handler:           middlewares.Hook(router, mws...),
		ReadHeaderTimeout: config.ReadHeaderTimeout,
		ReadTimeout:       config.ReadTimeout,
		WriteTimeout:      config.WriteTimeout,
		IdleTimeout:       config.IdleTimeout,
	}

	logrus.WithField("config", fmt.Sprintf("%+v", config)).Debug("Start REST API server")

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		logrus.WithError(err).WithField("endpoint", config.Endpoint).Fatal("Failed to serve HTTP server")
	}
}

func newCorsMiddleware(origins []string) gin.HandlerFunc {
	conf := cors.DefaultConfig()
	conf.AllowMethods = append(conf.AllowMethods, "OPTIONS")
	conf.AllowHeaders = append(conf.AllowHeaders, "*")

	if len(origins) == 0 {
		conf.AllowAllOrigins = true
	} else {
		conf.AllowOrigins = origins
	}

	return cors.New(conf)
}
