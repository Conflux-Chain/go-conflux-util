package config

import (
	"context"
	"sync"

	"github.com/Conflux-Chain/go-conflux-util/alert"
	"github.com/Conflux-Chain/go-conflux-util/log"
	"github.com/Conflux-Chain/go-conflux-util/metrics"
	"github.com/Conflux-Chain/go-conflux-util/pprof"
	"github.com/Conflux-Chain/go-conflux-util/viper"
)

// MustInitDefault initializes with "APP" env var prefix and try to load configs from "config.xxx" file.
// It shares MustInit's initialization and failure contracts.
func MustInitDefault() {
	MustInit("APP")
}

// MustInit performs the necessary initializations for the application, particularly
// by loading configuration settings from files or environment variables into `Viper`,
// setting up metrics, alerts, logging, and pprof.
// Call exactly one MustInit* entry point once during process startup, before using
// these components concurrently. Repeated initialization and hot reload are unsupported.
//
// Important: The order in which initializations are performed is critical due to
// dependencies between components.
//
// Parameters:
//   - viperEnvPrefix : The prefix used for environment variables that `Viper` should consider
//     while initializing configurations.
//   - configPath : The path to the configuration file. If not provided, search for the default
//     config file under current directory or `config` subfolder.
//
// Configuration loading/decoding, alert/log initialization, and pprof listening
// errors are logged with Fatal and exit the process; recover cannot catch these exits.
// Invalid component configuration may instead panic. Background startup errors
// may only be logged, so returning does not guarantee that every component is ready.
func MustInit(viperEnvPrefix string, configPath ...string) {
	// Delegates to the shared initialization logic with no context for graceful shutdown.
	mustInit(viperEnvPrefix, nil, nil, configPath...)
}

// MustInitWithCtx performs MustInit's initialization and connects configured
// asynchronous log-alert workers to ctx and wg. It shares MustInit's startup-only
// and failure contracts. Metrics reporting and pprof retain process lifetime.
//
// Both ctx and wg must be non-nil to enable worker shutdown handling; otherwise,
// logging is initialized without it. After initialization returns, stop log
// producers, cancel ctx, and wait on wg. Worker registration calls wg.Add internally;
// callers should not add a count for this initialization. Waiting covers only the
// registered workers and their best-effort queue drain, not all initialized services.
//
// Parameters:
//   - ctx: The cancellation context for asynchronous log-alert workers.
//   - wg: The wait group tracking those workers and their queue drain.
//   - viperEnvPrefix : The prefix used for environment variables that `Viper` should consider
//     while initializing configurations.
//   - configPath : The path to the configuration file. If not provided, search for the default
//     config file under current directory or `config` subfolder.
func MustInitWithCtx(
	ctx context.Context, wg *sync.WaitGroup, viperEnvPrefix string, configPath ...string) {
	mustInit(viperEnvPrefix, ctx, wg, configPath...)
}

// mustInit is the internal function responsible for the core initialization steps.
// It consolidates the setup of `Viper`, metrics, alerts, and logging, adapting the logging
// setup based on whether a worker cancellation context and wait group are provided.
//
// Important: The order in which initializations are performed is critical due to
// dependencies between components.
func mustInit(
	viperEnvPrefix string, ctx context.Context, wg *sync.WaitGroup, configPath ...string) {
	// Initialize `Viper` to read configurations from a file or environment variables.
	// The provided prefix is used to match and bind environment variables to config keys.
	viper.MustInit(viperEnvPrefix, configPath...)

	// Initialize metrics collection based on the configurations loaded into `Viper`.
	// Metrics are typically used for monitoring application performance.
	metrics.MustInitFromViper()

	// Initialize alerting systems using configurations from `Viper`.
	// Alerts are crucial for notifying about application errors or important events.
	alert.MustInitFromViper()

	// Initialize the logging system with configurations fetched via Viper.
	// Logging setup depends on alert initialization since it might use alerting channels.
	// The context and wait group apply only to configured asynchronous log-alert workers.
	if ctx != nil && wg != nil {
		log.MustInitWithCtxFromViper(ctx, wg)
	} else {
		log.MustInitFromViper()
	}

	pprof.MustInitFromViper()
}
