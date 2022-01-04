package viper

import (
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var envKeyPrefix string // environment variable prefix

// MustInit inits viper with provided env var prefix.
//
// Note that it will panic and exit if any error happens.
func MustInit(envPrefix string) {
	envKeyPrefix = strings.ToUpper(envPrefix + "_")

	// Read system enviroment prefixed variables.
	// eg., CFX_LOG_LEVEL will override "log.level" config item from config file.
	viper.AutomaticEnv()
	viper.SetEnvPrefix(envPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file from current directory or under config folder.
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	if err := viper.ReadInConfig(); err != nil {
		logrus.WithError(err).Fatal("Failed to read config to initialize viper")
	}

	logrus.WithField("configs", viper.GetViper().AllSettings()).Info("viper initialized")
}
