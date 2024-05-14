package log

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/Conflux-Chain/go-conflux-util/log/hook"
	viperUtil "github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/ethereum/go-ethereum/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// LoggingConfig logging configuration such as log level etc.,
type LoggingConfig struct {
	Level        string      `default:"info"` // logging level
	ForceColor   bool        // helpful on windows
	DisableColor bool        // helpful to output logs in file
	AlertHook    hook.Config // alert hooking configurations
}

// MustInitFromViper initializes the logging system using configurations from viper.
//
// Precondition:
//   - Viper must be initialized with appropriate configurations before calling this function.
//
// Panics:
//   - This function will panic if it encounters any errors during initialization.
func MustInitFromViper() {
	var conf LoggingConfig
	viperUtil.MustUnmarshalKey("log", &conf)

	MustInit(conf)
}

// MustInitWithCtxFromViper performs the similar initializations as `MustInitFromViper` with
// support for graceful shutdown by accepting a context and a wait group.
//
// Parameters:
//   - ctx: The context for graceful shutdown handling.
//   - wg: The wait group to track goroutines for shutdown synchronization.
func MustInitWithCtxFromViper(ctx context.Context, wg *sync.WaitGroup) {
	var conf LoggingConfig
	viperUtil.MustUnmarshalKey("log", &conf)

	MustInitWithCtx(ctx, wg, conf)
}

// MustInit sets up the logging system according to the provided LoggingConfig and log level.
// It configures the log level, adds an alert hook, sets a text formatter, and adapts the logger
// for Geth compatibility.
// In case of any error during initialization, this function will panic.
func MustInit(conf LoggingConfig) {
	mustInit(conf, nil, nil)
}

// MustInitWithContext performs the similiar initializations as `MustInit` with support for
// graceful shutdown by accepting a context and a wait group.
func MustInitWithCtx(ctx context.Context, wg *sync.WaitGroup, conf LoggingConfig) {
	mustInit(conf, ctx, wg)
}

// mustInit initializes the logging system with the provided configuration and sets up an alert hook.
// It supports graceful shutdown by optionally using a context and wait group for the alert hook registration.
func mustInit(conf LoggingConfig, ctx context.Context, wg *sync.WaitGroup) {
	// Parse the log level string from the configuration into a logrus.Level.
	// If parsing fails, log the error along with the attempted level and terminate the application.
	level, err := logrus.ParseLevel(conf.Level)
	if err != nil {
		logrus.WithError(err).WithField("level", conf.Level).Fatal("Failed to parse log level")
	}
	logrus.SetLevel(level) // Set the parsed log level.

	// Attempt to add an alert hook as configured.
	if err := hook.AddAlertHook(ctx, wg, conf.AlertHook); err != nil {
		logrus.WithError(err).Fatal("Failed to add alert hook")
	}

	// Configure the log formatter to use a text format with a full timestamp.
	formatter := &logrus.TextFormatter{
		FullTimestamp: true,
	}

	// Adjust the color settings of the formatter based on the configuration.
	if conf.DisableColor {
		formatter.DisableColors = true
	} else if conf.ForceColor {
		formatter.ForceColors = true
	}

	// Apply the configured formatter to the logger.
	logrus.SetFormatter(formatter)

	// Adapt the logger for use with Geth that uses a custom logging mechanism.
	adaptGethLogger()

	// Log a debug message indicating successful initialization along with the effective configuration.
	logrus.WithField("config", fmt.Sprintf("%+v", conf)).Debug("Log initialized")
}

// adaptGethLogger adapt geth logger to work with logrus.
func adaptGethLogger() {
	formatter := log.TerminalFormat(false)

	// Mapping from geth log level to logrus level
	logrusLevelsMap := map[log.Lvl]logrus.Level{
		log.LvlCrit:  logrus.FatalLevel,
		log.LvlError: logrus.ErrorLevel,
		// Geth warn logging is little bit verbose,
		// adapt it to logrus info level.
		log.LvlWarn:  logrus.InfoLevel,
		log.LvlInfo:  logrus.InfoLevel,
		log.LvlDebug: logrus.DebugLevel,
		log.LvlTrace: logrus.TraceLevel,
	}

	log.Root().SetHandler(log.FuncHandler(func(r *log.Record) error {
		logLvl, ok := logrusLevelsMap[r.Lvl]
		if !ok {
			return errors.New("unsupported geth log level")
		}

		// only logging above mapped logrus level
		if logLvl <= logrus.GetLevel() {
			logStr := string(formatter.Format(r))
			abbrStr := logStr

			firstLineEnd := strings.IndexRune(logStr, '\n')
			if firstLineEnd > 0 { // extract first line as abstract
				abbrStr = logStr[:firstLineEnd]
			}

			logrus.WithField("gethWrappedLogs", logStr).Log(logLvl, abbrStr)
		}

		return nil
	}))
}

// BindFlags binds logging relevant flags for specified command.
func BindFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("log-level", "info", "The logging level (trace|debug|info|warn|error|fatal|panic)")
	viper.BindPFlag("log.level", cmd.Flag("log-level"))

	cmd.PersistentFlags().Bool("log-force-color", false, "Force colored logs")
	viper.BindPFlag("log.forceColor", cmd.Flag("log-force-color"))

	cmd.PersistentFlags().Bool("log-disable-color", false, "Disable colored logs")
	viper.BindPFlag("log.disableColor", cmd.Flag("log-disable-color"))
}
