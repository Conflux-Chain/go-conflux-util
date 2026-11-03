// Package pprof starts an optional diagnostic HTTP server.
// Importing it also imports net/http/pprof, which registers handlers on
// http.DefaultServeMux even when this server is disabled. Other HTTP servers
// should use their own handlers to avoid exposing those diagnostic routes.
package pprof

import (
	"net"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Enabled  bool
	Endpoint string `default:"127.0.0.1:6060"`
}

// MustInitFromViper loads the "pprof" configuration, applies defaults, and calls
// MustInit. Configuration loading failures are logged with Fatal and exit the process.
func MustInitFromViper() {
	var config Config
	viper.MustUnmarshalKey("pprof", &config)
	MustInit(config)
}

// MustInit starts a background diagnostic server when Enabled is true.
// It uses config as-is; default tags are not applied here. Call it once during
// process initialization. Listening failures, including an occupied endpoint,
// are logged with Fatal and exit the process. The server runs for the lifetime
// of the process and exposes no graceful shutdown entry point.
func MustInit(config Config) {
	if !config.Enabled {
		return
	}

	listener, err := net.Listen("tcp", config.Endpoint)
	if err != nil {
		logrus.WithError(err).WithField("config", config).Fatal("Failed to listen for pprof service")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /debug/pprof/", pprof.Index)
	mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)

	// Leave read/write timeouts unset to allow long captures and profile downloads.
	server := http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	logrus.WithField("config", config).Info("Starting pprof service")

	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			logrus.WithError(err).WithField("config", config).Error("Failed to serve pprof")
		}
	}()
}
