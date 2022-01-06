package config

import (
	"github.com/Conflux-Chain/go-conflux-util/log"
	"github.com/Conflux-Chain/go-conflux-util/viper"
)

// MustInit inits settings especially by loading configs from file or env var
// to viper before using any utility.
//
// Note that it will panic and exit if any error happens.
func MustInit(viperEnvPrefix string) {
	// init viper from config file or env var
	viper.MustInit(viperEnvPrefix)
	// init logging from viper
	log.MustInitFromViper()
}
