package log

import (
	stdLog "log"
	"os"
	"strings"
	"time"

	viperutil "github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	gormlogger "gorm.io/gorm/logger"
)

// LoggingConfig logging configuration such as log level etc.,
type LoggingConfig struct {
	Level string `default:"info"` // logging level
}

// MustInitFromViper inits logging from viper settings and adapts Geth logger.
//
// Note that viper must be initilized before this, and it will panic
// and exit if any error happens.
func MustInitFromViper() {
	var conf LoggingConfig
	viperutil.MustUnmarshalKey("log", &conf)

	level, err := logrus.ParseLevel(conf.Level)
	if err != nil {
		logrus.WithError(err).
			WithField("logLevel", conf.Level).
			Fatal("Invalid log level parsed from viper config")
	}

	Init(level)
}

// Init inits logging with specified log level
func Init(level logrus.Level) {
	logrus.SetLevel(level)

	// adapt geth logger
	adaptGethLogger()

	// adapt gorm logger
	adaptGormLogger()
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

// adaptGormLogger adapt gorm logger to work with logrus.
func adaptGormLogger() {
	// gorm logs info level is somewhat verbose, only turn it on for logrus
	// trace level.
	gLogLevel := gormlogger.Error
	if logrus.IsLevelEnabled(logrus.TraceLevel) {
		gLogLevel = gormlogger.Info
	} else if logrus.IsLevelEnabled(logrus.WarnLevel) {
		gLogLevel = gormlogger.Warn
	}

	// customize the default gorm logger
	gormlogger.Default = gormlogger.New(
		// io writer
		stdLog.New(os.Stdout, "\r\n", stdLog.LstdFlags),
		gormlogger.Config{
			// slow SQL threshold (200ms)
			SlowThreshold: time.Millisecond * 200,
			// log level
			LogLevel: gLogLevel,
			// never logging on ErrRecordNotFound error, otherwise logs may grow exploded
			IgnoreRecordNotFoundError: true,
			// use colorful print
			Colorful: true,
		},
	)
}
