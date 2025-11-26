package viper

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
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

func reset() {
	os.Clearenv()
	viper.Reset()
	initEnv("CFX")
}

func TestRawUnmarshalConfig(t *testing.T) {
	reset()

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
	reset()

	// init env
	os.Setenv("CFX_FOO_BAR_VAL2", "env2")
	os.Setenv("CFX_FOO_BAR_VAL4", "env4")

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
	reset()

	// init env
	os.Setenv("CFX_FOO_BAR_VAL2", "env2")
	os.Setenv("CFX_FOO_BAR_VAL4", "env4")

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
