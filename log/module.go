package log

import "github.com/sirupsen/logrus"

var moduleLevels = map[string]logrus.Level{}

func mustInitModuleLevels(modules map[string]string) {
	for k, v := range modules {
		level, err := logrus.ParseLevel(v)
		if err != nil {
			logrus.WithError(err).WithField("level", v).Fatal("Failed to parse log level")
		}

		moduleLevels[k] = level
	}
}

// WithModule returns a logger with module name and specific log level.
func WithModule(module string) *logrus.Entry {
	moduleLogger := logrus.WithField("module", module)

	if level, ok := moduleLevels[module]; ok {
		moduleLogger.Logger.SetLevel(level)
	}

	return moduleLogger
}
