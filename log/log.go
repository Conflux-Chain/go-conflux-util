package log

import (
	"fmt"
	"strings"

	"github.com/Conflux-Chain/go-conflux-util/log/hook"
	viperUtil "github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// LoggingConfig logging configuration such as log level etc.,
type LoggingConfig struct {
	Level        string `default:"info"` // logging level
	ForceColor   bool   // helpful on windows
	DisableColor bool   // helpful to output logs in file

	// logrus level hooked for alert notification
	AlertHookLevels []string `default:"[warn,error,fatal]"`
}

// MustInitFromViper inits logging from viper settings and adapts Geth logger.
//
// Note that viper must be initilized before this, and it will panic
// and exit if any error happens.
func MustInitFromViper() {
	var conf LoggingConfig
	viperUtil.MustUnmarshalKey("log", &conf)

	MustInit(conf)
}

// Init inits logging with specified log level
func MustInit(conf LoggingConfig) {
	// parse logging level
	level, err := logrus.ParseLevel(conf.Level)
	if err != nil {
		logrus.WithError(err).WithField("level", conf.Level).Fatal("Failed to parse log level")
	}
	logrus.SetLevel(level)

	// hook alert logging levels
	var hookLvls []logrus.Level
	for _, lvlStr := range conf.AlertHookLevels {
		lvl, err := logrus.ParseLevel(lvlStr)
		if err != nil {
			logrus.WithError(err).WithField("level", lvlStr).Fatal("Failed to parse log level for alert hooking")
		}
		hookLvls = append(hookLvls, lvl)
	}

	if len(hookLvls) > 0 {
		hook.AddDingTalkAlertHook(hookLvls)
	}

	// set text formtter
	formatter := &logrus.TextFormatter{
		FullTimestamp: true,
	}

	if conf.DisableColor {
		formatter.DisableColors = true
	} else if conf.ForceColor {
		formatter.ForceColors = true
	}

	logrus.SetFormatter(formatter)

	// adapt geth logger
	adaptGethLogger()

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
