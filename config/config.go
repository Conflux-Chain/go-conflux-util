package config

import (
	"github.com/Conflux-Chain/go-conflux-util/alert"
	"github.com/Conflux-Chain/go-conflux-util/log"
	"github.com/Conflux-Chain/go-conflux-util/metrics"
	"github.com/Conflux-Chain/go-conflux-util/viper"
)

// MustInit inits settings especially by loading configs from file or env var
// to viper etc., to prepare using any utility.
//
// Note that it will panic and exit if any error happens.
func MustInit(viperEnvPrefix string) {
	// NOTE, INITIALIZATION ORDER IS IMPORTANT!

	// init viper from config file or env var
	viper.MustInit(viperEnvPrefix)

	// init metrics from viper
	metrics.MustInitFromViper()

	// init alert from viper
	alert.MustInitFromViper()

	// init logging from viper which depends on alert initialization
	log.MustInitFromViper()
}
