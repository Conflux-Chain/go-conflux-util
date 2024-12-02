package hook

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"

	"gopkg.in/natefinch/lumberjack.v2"
)

type FileConfig struct {
	// Groups defines the log groups and their associated logging file configuration.
	Groups map[string]FileGroup

	// Async configures the behavior of the asynchronous worker for handling log alerts.
	Async AsyncOption
}

type FileGroup struct {
	Levels   []string       // Log levels included in this group
	Location string         // File path for the log group
	Rotation RotationConfig // Log rotation settings
}

// RotationConfig defines the rotation settings for log files.
type RotationConfig struct {
	MaxSize    int  `default:"100"` // Maximum size of the log file in MB before rotation
	MaxBackups int  // Maximum number of backup files to retain (0 to keep all)
	MaxAge     int  `default:"30"`   // Maximum age of log files before deletion (e.g., 30 days)
	Compress   bool `default:"true"` // Whether to compress rotated log files
}

func newLumberjackLogger(grpConf FileGroup) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   grpConf.Location,
		MaxSize:    grpConf.Rotation.MaxSize,
		MaxBackups: grpConf.Rotation.MaxBackups,
		MaxAge:     grpConf.Rotation.MaxAge,
		Compress:   grpConf.Rotation.Compress,
	}
}

// NewFileHook constructor to new FileHook instance.
func NewFileHook(conf FileConfig, formatter logrus.Formatter) (*lfshook.LfsHook, error) {
	var defaultLogger *lumberjack.Logger
	writerMap := make(lfshook.WriterMap)

	for grp, grpConf := range conf.Groups {
		if strings.EqualFold(grp, "default") {
			defaultLogger = newLumberjackLogger(grpConf)
			continue
		}
		if len(grpConf.Levels) == 0 {
			return nil, errors.Errorf("no log levels defined for group %s", grp)
		}

		logger := newLumberjackLogger(grpConf)
		for _, l := range grpConf.Levels {
			level, err := logrus.ParseLevel(l)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to parse log level")
			}
			if _, ok := writerMap[level]; ok {
				return nil, errors.Errorf("duplicate log level %s in group %s", l, grp)
			}
			writerMap[level] = logger
		}
	}

	hook := lfshook.NewHook(writerMap, formatter)
	if defaultLogger != nil {
		hook.SetDefaultWriter(defaultLogger)
	}
	return hook, nil
}
