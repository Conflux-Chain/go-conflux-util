package viper

import (
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	envKeyPrefix string // environment variable prefix
)

func initEnv(envPrefix string) {
	envKeyPrefix = strings.ToUpper(envPrefix + "_")

	// Read system enviroment prefixed variables.
	// eg., CFX_LOG_LEVEL will override "log.level" config item from config file.
	viper.AutomaticEnv()
	viper.SetEnvPrefix(envPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
}

// MustInit inits viper with provided env var prefix.
//
// Note that it will panic and exit if any error happens.
func MustInit(envPrefix string, configPath ...string) {
	logger := logrus.WithField("envPrefix", envPrefix)
	initEnv(envPrefix)

	if len(configPath) > 0 {
		logger = logger.WithField("customConfig", configPath[0])
		viper.SetConfigFile(configPath[0])
	} else {
		// Read config file from current directory or under config folder.
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
		viper.AddConfigPath("./config")
	}

	if err := viper.ReadInConfig(); err != nil {
		logger.WithError(err).Fatal("Failed to read config to initialize viper")
	}

	logger.Debug("Viper initialized")
}
