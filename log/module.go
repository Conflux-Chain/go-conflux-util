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
//
// Note, DO NOT use dot, hyphen, underline or some other special characters in key name.
// Otherwise, there may be unexpected parse error in different config files. E.g.
//
//   - a.b.c: dot is not allowed as key name in .env file.
//   - a-b-c: hyphen is not allowed as key name in .env file.
//   - a_b_c: underline usually indicates a sub-config in .env file.
//
// On the other hand, viper will convert all keys to lowercase. As a result, it is highly
// recommended to use lowercase module name.
func WithModule(module string) *logrus.Entry {
	moduleLogger := logrus.WithField("module", module)

	if level, ok := moduleLevels[module]; ok {
		moduleLogger.Logger.SetLevel(level)
	}

	return moduleLogger
}
