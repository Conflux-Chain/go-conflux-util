package config

import (
	"context"
	"sync"

	"github.com/Conflux-Chain/go-conflux-util/alert"
	"github.com/Conflux-Chain/go-conflux-util/log"
	"github.com/Conflux-Chain/go-conflux-util/metrics"
	"github.com/Conflux-Chain/go-conflux-util/viper"
)

// MustInit performs the necessary initializations for the application, particularly
// by loading configuration settings from files or environment variables into `Viper`,
// setting up metrics, alerts, and logging systems. This function is designed to be
// used at the application's startup phase.
//
// Important: The order in which initializations are performed is critical due to
// dependencies between components.
//
// Parameters:
//   - viperEnvPrefix : The prefix used for environment variables that `Viper` should consider
//     while initializing configurations.
//
// Panics:
//   - If any part of the initialization fails, this function will panic, causing the application
//     to terminate abruptly.
func MustInit(viperEnvPrefix string) {
	// Delegates to the shared initialization logic with no context for graceful shutdown.
	mustInit(viperEnvPrefix, nil, nil)
}

// MustInitWithContext carries out the same initializations as `MustInit` except for support for
// graceful shutdown by accepting a context and a wait group.
//
// Parameters:
//   - ctx: The context for graceful shutdown handling.
//   - wg: The wait group to track goroutines for shutdown synchronization.
//   - viperEnvPrefix : The prefix used for environment variables that `Viper` should consider
//     while initializing configurations.
func MustInitWithCtx(ctx context.Context, wg *sync.WaitGroup, viperEnvPrefix string) {
	mustInit(viperEnvPrefix, ctx, wg)
}

// mustInit is the internal function responsible for the core initialization steps.
// It consolidates the setup of `Viper`, metrics, alerts, and logging, adapting the logging
// setup based on whether graceful shutdown context and wait group are provided.
//
// Important: The order in which initializations are performed is critical due to
// dependencies between components.
func mustInit(viperEnvPrefix string, ctx context.Context, wg *sync.WaitGroup) {
	// Initialize `Viper` to read configurations from a file or environment variables.
	// The provided prefix is used to match and bind environment variables to config keys.
	viper.MustInit(viperEnvPrefix)

	// Initialize metrics collection based on the configurations loaded into `Viper`.
	// Metrics are typically used for monitoring application performance.
	metrics.MustInitFromViper()

	// Initialize alerting systems using configurations from `Viper`.
	// Alerts are crucial for notifying about application errors or important events.
	alert.MustInitFromViper()

	// Initialize the logging system with configurations fetched via Viper.
	// Logging setup depends on alert initialization since it might use alerting channels.
	// Additionally, logging setup accepts a context and wait group to ensure logs are handled
	// properly during shutdown.
	if ctx != nil && wg != nil {
		log.MustInitWithCtxFromViper(ctx, wg)
	} else {
		log.MustInitFromViper()
	}
}
