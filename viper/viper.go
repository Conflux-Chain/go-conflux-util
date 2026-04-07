package viper

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	envKeyPrefix string // environment variable prefix
)

func initEnv(envPrefix string) {
	if len(envPrefix) > 0 {
		envKeyPrefix = strings.ToUpper(envPrefix + "_")
	}

	// Read system environment prefixed variables.
	// eg., CFX_LOG_LEVEL will override "log.level" config item from config file.
	viper.AutomaticEnv()
	viper.SetEnvPrefix(envPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
}

// MustInit inits viper with provided env var prefix (e.g. "CFX" or empty string) and optional config files, e.g.
// config.yaml, config-prod.yaml, config-local.yaml.
//
// If no config file specified, viper will search for config.xxx file by default.
//
// Note that it will panic and exit if failed to read from config files.
func MustInit(envPrefix string, configFiles ...string) {
	logger := logrus.WithField("envPrefix", envPrefix)

	// load .env file if exists
	if err := godotenv.Load(); err != nil {
		if !os.IsNotExist(err) {
			logrus.WithError(err).Fatal("Failed to load .env file")
		}
	}

	// enable environment variables by default
	initEnv(envPrefix)

	if len(configFiles) > 0 {
		// read config from specified files
		for _, v := range configFiles {
			viper.SetConfigFile(v)

			if err := viper.MergeInConfig(); err != nil {
				logger.WithError(err).WithField("file", v).Fatal("Failed to read config file")
			}
		}

		logger = logger.WithField("files", configFiles)
	} else {
		// read from config.xxx file if exists
		viper.AddConfigPath(".")
		viper.AddConfigPath("config")

		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				logger.WithError(err).Fatal("Failed to read config file")
			}
		}
	}

	logger.Debug("Viper initialized")
}
