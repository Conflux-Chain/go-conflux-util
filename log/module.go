package log

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var moduleLoggers map[string]*logrus.Logger

// mustInitLogLevels initializes log levels in format of "level,module1=level1,module2=level2", where module log level is optional.
func mustInitLogLevels(levels string) map[string]logrus.Level {
	defaultLevel, moduleLevelMap, err := parseLogLevels(levels)
	if err != nil {
		logrus.WithError(err).WithField("levels", levels).Fatal("Failed to parse log levels")
	}

	logrus.SetLevel(defaultLevel)

	return moduleLevelMap
}

// parseLogLevels parses log levels in format of "level,module1=level1,moduel2=level2", where module log level is optional.
func parseLogLevels(levels string) (defaultLogLevel logrus.Level, moduleLogLevels map[string]logrus.Level, err error) {
	if len(levels) == 0 {
		return logrus.InfoLevel, nil, errors.New("Log level is empty")
	}

	modules := strings.Split(levels, ",")

	// index 0 is the default log level
	if defaultLogLevel, err = logrus.ParseLevel(modules[0]); err != nil {
		return logrus.InfoLevel, nil, errors.WithMessagef(err, "Failed to parse the default log level: %v", modules[0])
	}

	moduleLogLevels = make(map[string]logrus.Level)

	for _, v := range modules[1:] {
		fields := strings.SplitN(v, "=", 2)
		if len(fields) != 2 {
			return logrus.InfoLevel, nil, errors.Errorf("Invalid format of module log level: %v", v)
		}

		level, err := logrus.ParseLevel(fields[1])
		if err != nil {
			return logrus.InfoLevel, nil, errors.WithMessagef(err, "Failed to parse log level, module = %v, level = %v", fields[0], fields[1])
		}

		moduleLogLevels[fields[0]] = level
	}

	return
}

// initModuleLoggers snapshots the configured standard logger before concurrent logging starts.
func initModuleLoggers(levels map[string]logrus.Level) {
	base := logrus.StandardLogger()
	loggers := make(map[string]*logrus.Logger, len(levels))
	for module, level := range levels {
		// Use a new logger so its mutex, entry pool and level are independent.
		logger := logrus.New()
		logger.Out = base.Out
		logger.Formatter = base.Formatter
		logger.ReportCaller = base.ReportCaller
		logger.ExitFunc = base.ExitFunc
		logger.BufferPool = base.BufferPool
		logger.SetLevel(level)
		// Share hook instances, but keep each logger's hook registry independent.
		for hookLevel, hooks := range base.Hooks {
			logger.Hooks[hookLevel] = append([]logrus.Hook(nil), hooks...)
		}
		loggers[module] = logger
	}
	moduleLoggers = loggers
}

// WithModule returns an entry with the module name and its configured log level.
// Explicitly configured modules reuse independent loggers created during initialization;
// later standard logger configuration changes are not propagated to them.
// Unconfigured modules use the standard logger. Initialize logging before creating module entries.
//
// Note, the module name should not contain delimiters "=" and ",".
// Otherwise, the behavior is unknown.
func WithModule(module string) *logrus.Entry {
	if logger, ok := moduleLoggers[module]; ok {
		return logger.WithField("module", module)
	}
	return logrus.WithField("module", module)
}
