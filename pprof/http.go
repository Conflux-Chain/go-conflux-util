package pprof

import (
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
	Init(config)
}

// Init initializes pprof for the given config.
func Init(config Config) {
	if !config.Enabled {
		return
	}

	logrus.WithField("config", config).Info("Starting pprof service")

	go http.ListenAndServe(config.Endpoint, nil)
}
