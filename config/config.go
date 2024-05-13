package config

import (
	"github.com/Conflux-Chain/go-conflux-util/alert"
	"github.com/Conflux-Chain/go-conflux-util/graceful"
	"github.com/Conflux-Chain/go-conflux-util/log"
	"github.com/Conflux-Chain/go-conflux-util/metrics"
	"github.com/Conflux-Chain/go-conflux-util/viper"
)

// MustInit performs the necessary initializations for the application, particularly
// by loading configuration settings from files or environment variables into Viper,
// setting up metrics, alerts, and logging systems. This function is designed to be
// used at the application's startup phase.
//
// Important: The order in which initializations are performed is critical due to
// dependencies between components.
//
// Parameters:
//   - viperEnvPrefix : The prefix used for environment variables that `Viper` should consider
//     while initializing configurations.
//   - handler: The pointer to an optional `ShutdownHandler` instances to be passed to the logger
//     for graceful shutdown handling.
//
// Panics:
//   - If any part of the initialization fails, this function will panic, causing the application
//     to terminate abruptly.
func MustInit(viperEnvPrefix string, sh ...*graceful.ShutdownHandler) {
	// Initialize Viper to read configurations from a file or environment variables.
	// The provided prefix is used to match and bind environment variables to config keys.
	// If any error occurs during this process, Viper's MustInit will panic.
	viper.MustInit(viperEnvPrefix)

	// Initialize metrics collection based on the configurations loaded into Viper.
	// Metrics are typically used for monitoring application performance.
	// Any misconfiguration will cause a panic.
	metrics.MustInitFromViper()

	// Initialize alerting systems using configurations from Viper.
	// Alerts are crucial for notifying about application errors or important events.
	// Configuration errors here will also lead to a panic.
	alert.MustInitFromViper()

	// Initialize the logging system with configurations fetched via Viper.
	// Logging setup depends on alert initialization since it might use alerting channels.
	// Additionally, logging setup accepts graceful shutdown handlers to ensure logs are flushed
	// properly during shutdown.
	// Like previous steps, a failure in logging setup will result in a panic.
	log.MustInitFromViper(sh...)
}
