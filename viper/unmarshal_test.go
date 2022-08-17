package viper

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// Stub struct for testing
type logLimit struct {
	MaxLogs int `mapstructure:"maxLogs"`
}

type threshold struct {
	Tags []string `mapstructure:"tags"`
	Log  logLimit `mapstructure:"log"`
}

type pruneConfig struct {
	Name      string        `mapstructure:"name"`
	Eanbled   bool          `mapstructure:"enabled"`
	Interval  time.Duration `mapstructure:"interval"`
	Threshold threshold     `mapstructure:"threshold"`
}

func init() {
	initEnv("cfx")
}

func TestViperSub(t *testing.T) {
	os.Setenv("CFX_PRUNE_THRESHOLD_LOG_MAXLOGS", "1000")

	var jsonConf = []byte(`{"prune":{"name":"ptest1","enabled":true,"interval":"1s","threshold":{"tags":["block","prune"],"log":{"maxLogs":"2000"}}}}`)

	viper.SetConfigType("json")
	err := viper.ReadConfig(bytes.NewBuffer(jsonConf))
	assert.Nil(t, err)

	{ // test viper.Sub(...) does not respect env var
		var pc pruneConfig
		err = viper.Sub("prune").Unmarshal(&pc)
		assert.Nil(t, err)
		assert.NotEqualValues(t, 1000, pc.Threshold.Log.MaxLogs)
	}

	var pc pruneConfig
	assert.Nil(t, UnmarshalKey("prune", &pc))

	{ // test ViperSub(...) works with env var for type int
		assert.EqualValues(t, 1000, pc.Threshold.Log.MaxLogs)

		os.Setenv("CFX_PRUNE_THRESHOLD_LOG_MAXLOGS", "5000")

		var pc2 pruneConfig
		assert.Nil(t, UnmarshalKey("prune", &pc2))

		assert.EqualValues(t, 5000, pc2.Threshold.Log.MaxLogs)

		os.Setenv("CFX_PRUNE_THRESHOLD_LOG_MAXLOGS", "15000")

		var llc logLimit
		assert.Nil(t, UnmarshalKey("prune.threshold.log", &llc))

		assert.EqualValues(t, 15000, llc.MaxLogs)
	}

	{ // test ViperSub(...) works with env var for type bool
		assert.True(t, pc.Eanbled)

		os.Setenv("CFX_PRUNE_ENABLED", "false")

		var pc2 pruneConfig
		assert.Nil(t, UnmarshalKey("prune", &pc2))

		assert.False(t, pc2.Eanbled)
	}

	{ // test ViperSub(...) works with env var for type string
		assert.Equal(t, "ptest1", pc.Name)

		os.Setenv("CFX_PRUNE_NAME", "ptest2")

		var pc2 pruneConfig
		assert.Nil(t, UnmarshalKey("prune", &pc2))

		assert.Equal(t, "ptest2", pc2.Name)
	}

	{ // test ViperSub(...) works with env var for type time.Duration
		assert.Equal(t, time.Second, pc.Interval)

		os.Setenv("CFX_PRUNE_INTERVAL", "5m")

		var pc2 pruneConfig
		assert.Nil(t, UnmarshalKey("prune", &pc2))

		assert.Equal(t, 5*time.Minute, pc2.Interval)
	}

	{ // test ViperSub(...) works with env var for type []string
		assert.ElementsMatch(t, pc.Threshold.Tags, []string{"block", "prune"})

		os.Setenv("CFX_PRUNE_THRESHOLD_TAGS", "tx delete")

		var pc2 pruneConfig
		assert.Nil(t, UnmarshalKey("prune", &pc2))

		assert.ElementsMatch(t, pc2.Threshold.Tags, []string{"tx delete"})

		var pc3 pruneConfig
		assert.Nil(t, UnmarshalKey("prune", &pc3, func(key string) (interface{}, bool) {
			if key == "prune.threshold.tags" {
				return viper.GetStringSlice(key), true
			}

			return nil, false
		}))

		assert.ElementsMatch(t, pc3.Threshold.Tags, []string{"tx", "delete"})
	}
}

func TestEnvWithoutConfig(t *testing.T) {
	os.Setenv("CFX_TESTCONFIG_ENABLED", "true")
	os.Setenv("CFX_TESTCONFIG_SUBCONFIG_ENABLED", "true")

	var conf struct {
		Enabled   bool
		SubConfig struct {
			Enabled bool
		}
	}

	MustUnmarshalKey("testConfig", &conf)
	assert.True(t, conf.Enabled)
	assert.True(t, conf.SubConfig.Enabled)
}

func TestEnvConfigMixed(t *testing.T) {
	var conf struct {
		Val0 string
		Val1 string `default:"v1"`
		Val2 string `default:"v2"` // env
		Val3 string `default:"v3"` // yml
		Val4 string `default:"v4"` // yml + env: env will overwrite yml
	}

	os.Setenv("CFX_TESTCONFIG_VAL2", "env2")
	os.Setenv("CFX_TESTCONFIG_VAL4", "env4")

	var jsonConf = []byte(`
testConfig:
  val3: yml3
  val4: yml4
`)
	viper.SetConfigType("yaml")
	assert.NoError(t, viper.ReadConfig(bytes.NewBuffer(jsonConf)))

	MustUnmarshalKey("testConfig", &conf)
	assert.Equal(t, "", conf.Val0)
	assert.Equal(t, "v1", conf.Val1)
	assert.Equal(t, "env2", conf.Val2)
	assert.Equal(t, "yml3", conf.Val3)
	assert.Equal(t, "env4", conf.Val4)
}

/*
 * For environment value, `UnmarshalKey` and `GetStringSlice` require different format.
 * - `UnmarshalKey` requires comma separated value, e.g. "a,b,c"
 * - `GetStringSlice` requires space separated value, e.g. "a b c"
 */
func TestEnvStringSlice(t *testing.T) {
	var conf struct {
		Names1 []string
		Names2 []string
	}

	os.Setenv("CFX_TESTCONFIG_NAMES1", "a,b,c")
	os.Setenv("CFX_TESTCONFIG_NAMES2", "a b c")

	MustUnmarshalKey("testConfig", &conf)
	assert.Equal(t, []string{"a", "b", "c"}, conf.Names1)
	assert.Equal(t, []string{"a b c"}, conf.Names2)

	assert.Equal(t, []string{"a,b,c"}, viper.GetStringSlice("testConfig.names1"))
	assert.Equal(t, []string{"a", "b", "c"}, viper.GetStringSlice("testConfig.names2"))
}
