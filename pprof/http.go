package pprof

import (
	"net"
	"net/http"
	_ "net/http/pprof"

	"github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Enabled  bool
	Endpoint string `default:":6060"`
}

// MustInitFromViper should initialize pprof from configuration.
func MustInitFromViper() {
	var config Config
	viper.MustUnmarshalKey("pprof", &config)
	MustInit(config)
}

// Init initializes pprof for the given config.
func MustInit(config Config) {
	if !config.Enabled {
		return
	}

	listener, err := net.Listen("tcp", config.Endpoint)
	if err != nil {
		logrus.WithError(err).WithField("config", config).Fatal("Failed to listen for pprof service")
	}

	logrus.WithField("config", config).Info("Starting pprof service")

	go http.Serve(listener, nil)
}
