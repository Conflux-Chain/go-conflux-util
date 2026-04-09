package log

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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
