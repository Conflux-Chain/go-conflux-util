package log

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLogLevels(t *testing.T) {
	// empty
	_, _, err := parseLogLevels("")
	assert.Error(t, err)

	// default level
	defaultLevel, moduleLevels, err := parseLogLevels("debug")
	assert.NoError(t, err)
	assert.Equal(t, logrus.DebugLevel, defaultLevel)
	assert.Empty(t, moduleLevels)

	// 1 module
	defaultLevel, moduleLevels, err = parseLogLevels("debug,aaa=debug")
	assert.NoError(t, err)
	assert.Equal(t, logrus.DebugLevel, defaultLevel)
	assert.Equal(t, map[string]logrus.Level{"aaa": logrus.DebugLevel}, moduleLevels)

	// 3 modules
	defaultLevel, moduleLevels, err = parseLogLevels("debug,a.b.c=debug,a-b-c=debug,a_b_c=debug")
	assert.NoError(t, err)
	assert.Equal(t, logrus.DebugLevel, defaultLevel)
	assert.Equal(t, map[string]logrus.Level{
		"a.b.c": logrus.DebugLevel,
		"a-b-c": logrus.DebugLevel,
		"a_b_c": logrus.DebugLevel,
	}, moduleLevels)

	// default level invalid
	_, _, err = parseLogLevels("foo")
	assert.Error(t, err)

	// module log level value missed
	_, _, err = parseLogLevels("debug,module1=debug,module2=,module3=debug")
	assert.Error(t, err)

	// module log level value invalid
	_, _, err = parseLogLevels("debug,module1=debug,module2=foo,module3=debug")
	assert.Error(t, err)
}

type moduleTestHook struct {
	messages []string
}

func (*moduleTestHook) Levels() []logrus.Level { return logrus.AllLevels }
func (h *moduleTestHook) Fire(entry *logrus.Entry) error {
	h.messages = append(h.messages, entry.Message)
	return nil
}

func preserveLoggingState(t *testing.T) {
	t.Helper()
	logger := logrus.StandardLogger()
	output, formatter, level := logger.Out, logger.Formatter, logger.GetLevel()
	reportCaller := logger.ReportCaller
	hooks := logger.ReplaceHooks(make(logrus.LevelHooks))
	modules := moduleLoggers
	t.Cleanup(func() {
		logger.SetOutput(output)
		logger.SetFormatter(formatter)
		logger.SetLevel(level)
		logger.SetReportCaller(reportCaller)
		logger.ReplaceHooks(hooks)
		moduleLoggers = modules
	})
}

func TestModuleLogLevelsAreIndependent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "modules.log")
	preserveLoggingState(t)
	alerts := &moduleTestHook{}
	logrus.AddHook(alerts)
	logrus.SetReportCaller(true)
	MustInit(LoggingConfig{
		Level: "info,rpc=debug,db=error", DisableColor: true,
		Output: OutputConfig{Type: "file", FilePath: path},
	})
	outputFile := logrus.StandardLogger().Out.(io.Closer)
	t.Cleanup(func() { assert.NoError(t, outputFile.Close()) })

	rpc := WithModule("rpc")
	db := WithModule("db")
	other := WithModule("other")
	assert.Equal(t, logrus.InfoLevel, logrus.GetLevel())
	assert.Equal(t, logrus.DebugLevel, rpc.Logger.GetLevel())
	assert.Equal(t, logrus.ErrorLevel, db.Logger.GetLevel())
	assert.Equal(t, logrus.InfoLevel, other.Logger.GetLevel())
	assert.NotSame(t, logrus.StandardLogger(), rpc.Logger)
	assert.NotSame(t, rpc.Logger, db.Logger)
	assert.Same(t, logrus.StandardLogger(), other.Logger)
	assert.Same(t, logrus.StandardLogger().Formatter, rpc.Logger.Formatter)
	rpcOnly := &moduleTestHook{}
	rpc.Logger.AddHook(rpcOnly)

	// Reuse the RPC entry after obtaining a DB entry with a different level.
	logrus.Debug("hidden-global-debug")
	logrus.Info("global-info")
	rpc.Debug("rpc-debug")
	rpc.Trace("hidden-rpc-trace")
	db.Warn("hidden-db-warn")
	db.Error("db-error")
	other.Debug("hidden-other-debug")
	other.Info("other-info")
	WithModule("rpc").Debug("rpc-again")

	expected := []string{"global-info", "rpc-debug", "db-error", "other-info", "rpc-again"}
	assert.Equal(t, expected, alerts.messages)
	assert.Equal(t, []string{"rpc-debug", "rpc-again"}, rpcOnly.messages)
	output, err := os.ReadFile(path)
	require.NoError(t, err)
	for _, message := range expected {
		assert.Contains(t, string(output), "msg="+message)
	}
	assert.NotContains(t, string(output), "hidden-")
	assert.Contains(t, string(output), "module=rpc")
	assert.Contains(t, string(output), "module=db")
	assert.Contains(t, string(output), "module=other")
	assert.Contains(t, string(output), "func=")
	assert.NotContains(t, string(output), "\x1b[")
	assert.Same(t, rpc.Logger, WithModule("rpc").Logger)
}

func TestModuleLoggerKeepsInitialConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "module.log")
	preserveLoggingState(t)
	originalHook := &moduleTestHook{}
	logrus.AddHook(originalHook)
	MustInit(LoggingConfig{
		Level: "info,rpc=debug", DisableColor: true,
		Output: OutputConfig{Type: "file", FilePath: path},
	})
	outputFile := logrus.StandardLogger().Out.(io.Closer)
	t.Cleanup(func() { assert.NoError(t, outputFile.Close()) })
	rpc := WithModule("rpc")

	var globalOutput bytes.Buffer
	globalHook := &moduleTestHook{}
	logrus.SetOutput(&globalOutput)
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.ErrorLevel)
	logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	logrus.AddHook(globalHook)

	rpc.Debug("rpc-debug")
	logrus.Error("global-error")
	output, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(output), "level=debug")
	assert.Contains(t, string(output), "msg=rpc-debug")
	assert.NotContains(t, string(output), "global-error")
	assert.Contains(t, globalOutput.String(), `"msg":"global-error"`)
	assert.NotContains(t, globalOutput.String(), "rpc-debug")
	assert.Equal(t, []string{"rpc-debug"}, originalHook.messages)
	assert.Equal(t, []string{"global-error"}, globalHook.messages)
	assert.Equal(t, logrus.DebugLevel, WithModule("rpc").Logger.GetLevel())
}
