package api

import (
	"fmt"
	"net/http"

	"github.com/Conflux-Chain/go-conflux-util/http/middlewares"
	"github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Endpoint string `default:":12345"`

	RecoveryDisabled bool
	CorsOrigins      []string
	Logger           bool
}

type RouteFactory func(router *gin.Engine)

func MustServeFromViper(factory RouteFactory, middlewares ...middlewares.Middleware) {
	var config Config
	viper.MustUnmarshalKey("api", &config)
	MustServe(config, factory, middlewares...)
}

func MustServe(config Config, factory RouteFactory, mws ...middlewares.Middleware) {
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
		Addr:    config.Endpoint,
		Handler: middlewares.Hook(router, mws...),
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

func Wrap(controller func(c *gin.Context) (interface{}, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := controller(c)
		if err != nil {
			ResponseError(c, err)
		} else {
			ResponseSuccess(c, result)
		}
	}
}
