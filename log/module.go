package log

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var moduleLevels map[string]logrus.Level

// mustInitLogLevels initializes log levels in format of "level,module1=level1,module2=level2", where module log level is optional.
func mustInitLogLevels(levels string) {
	defaultLevel, moduleLevelMap, err := parseLogLevels(levels)
	if err != nil {
		logrus.WithError(err).WithField("levels", levels).Fatal("Failed to parse log levels")
	}

	logrus.SetLevel(defaultLevel)
	moduleLevels = moduleLevelMap
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

// WithModule returns a logger with module name and specific log level.
//
// Note, the module name should not contain delimiters "=" and ",".
// Otherwise, the behavior is unknown.
func WithModule(module string) *logrus.Entry {
	moduleLogger := logrus.WithField("module", module)

	if level, ok := moduleLevels[module]; ok {
		moduleLogger.Logger.SetLevel(level)
	}

	return moduleLogger
}
