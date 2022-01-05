package log

import (
	"strings"

	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// InitFromViper inits logging from viper settings and adapts Geth logger.
//
// Note that viper must be initilized before, otherwise settings will not
// be loaded correctly.
func InitFromViper() {
	lvl := viper.GetString("log.level")
	level, err := logrus.ParseLevel(lvl)
	if err != nil {
		logrus.WithField("logLevel", lvl).WithError(err).Info(
			"Invalid log level parsed from viper config, use info level as default.",
		)

		level = logrus.InfoLevel
	}

	logrus.SetLevel(level)

	// adapt geth logger
	adaptGethLogger()
}

// AddDingTalkAlertHook adds logrus hook for DingTalk alert with specified log levels.
func AddDingTalkAlertHook(hookLevels []logrus.Level) {
	dingTalkAlertHook := NewDingTalkAlertHook(hookLevels)
	logrus.AddHook(dingTalkAlertHook)
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
