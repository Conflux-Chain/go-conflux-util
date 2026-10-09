package viper

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestConfig struct {
	Foo struct {
		Bar ValConfig
	}
}

type ValConfig struct {
	Val1 string `default:"val1"`
	Val2 string `default:"val2"`
	Val3 string `default:"val3"`
	Val4 string `default:"val4"`
}

// reset isolates the process-wide Viper and CFX environment used by these tests.
// Tests using this helper must not run in parallel.
func reset(t *testing.T) {
	t.Helper()

	// Save inherited test configuration without touching unrelated variables.
	inheritedEnv := make(map[string]string)
	for _, entry := range os.Environ() {
		key, value, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(strings.ToUpper(key), "CFX_") {
			inheritedEnv[key] = value
		}
	}

	// Register restoration before removing variables so failures still clean up.
	// Test-specific t.Setenv cleanups run first, in reverse registration order.
	t.Cleanup(func() {
		viper.Reset()
		envKeyPrefix = ""
		for key, value := range inheritedEnv {
			assert.NoError(t, os.Setenv(key, value), "restore environment variable %s", key)
		}
	})

	for key := range inheritedEnv {
		require.NoError(t, os.Unsetenv(key), "unset environment variable %s", key)
	}

	initEnv("CFX")
}

func TestRawUnmarshalConfig(t *testing.T) {
	reset(t)

	// init config
	viper.SetConfigType("yml")
	assert.NoError(t, viper.ReadConfig(bytes.NewBuffer([]byte(`
foo:
  bar:
    val3: yml3
    val4: yml4
`))))

	// global config
	var config TestConfig
	assert.NoError(t, viper.Unmarshal(&config))
	assert.Equal(t, ValConfig{"", "", "yml3", "yml4"}, config.Foo.Bar)

	// sub config
	var sub ValConfig
	assert.NoError(t, viper.UnmarshalKey("foo.bar", &sub))
	assert.Equal(t, ValConfig{"", "", "yml3", "yml4"}, sub)
}

func TestRawUnmarshalEnv(t *testing.T) {
	reset(t)

	// init env
	t.Setenv("CFX_FOO_BAR_VAL2", "env2")
	t.Setenv("CFX_FOO_BAR_VAL4", "env4")

	// global config - works bad to overwrite by env
	var config TestConfig
	assert.NoError(t, viper.Unmarshal(&config))
	// assert.Equal(t, ValConfig{"", "env2", "", "env4"}, config.Foo.Bar)
	assert.Equal(t, ValConfig{"", "", "", ""}, config.Foo.Bar)

	// sub config - works bad to overwrite by env
	var sub ValConfig
	assert.NoError(t, viper.UnmarshalKey("foo.bar", &sub))
	// assert.Equal(t, ValConfig{"", "env2", "", "env4"}, sub)
	assert.Equal(t, ValConfig{"", "", "", ""}, sub)
}

func TestRawUnmarshalMixed(t *testing.T) {
	reset(t)

	// init env
	t.Setenv("CFX_FOO_BAR_VAL2", "env2")
	t.Setenv("CFX_FOO_BAR_VAL4", "env4")

	// init config
	viper.SetConfigType("yml")
	assert.NoError(t, viper.ReadConfig(bytes.NewBuffer([]byte(`
foo:
  bar:
    val3: yml3
    val4: yml4
`))))

	// global config - works bad to overwrite by env
	var config TestConfig
	assert.NoError(t, viper.Unmarshal(&config))
	// assert.Equal(t, ValConfig{"", "env2", "yml3", "env4"}, config.Foo.Bar)
	assert.Equal(t, ValConfig{"", "", "yml3", "env4"}, config.Foo.Bar)

	// sub config - works bad to overwrite by env
	var sub ValConfig
	assert.NoError(t, viper.UnmarshalKey("foo.bar", &sub))
	// assert.Equal(t, ValConfig{"", "", "yml3", "env4"}, foo)
	assert.Equal(t, ValConfig{"", "", "yml3", "yml4"}, sub)
}
