package viper

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/viper/testutil"
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

func TestViperSub(t *testing.T) {
	reset()

	os.Setenv("CFX_PRUNE_THRESHOLD_LOG_MAXLOGS", "1000")

	var jsonConf = []byte(`{"prune":{"name":"ptest1","enabled":true,"interval":"1s","threshold":{"tags":["block","prune"],"log":{"maxLogs":"2000"}}}}`)

	viper.SetConfigType("json")
	assert.NoError(t, viper.ReadConfig(bytes.NewBuffer(jsonConf)))

	{ // test raw viper methods that does not respect env var
		var pc1 pruneConfig
		assert.NoError(t, viper.Sub("prune").Unmarshal(&pc1))
		assert.Equal(t, 2000, pc1.Threshold.Log.MaxLogs)

		var pc2 pruneConfig
		assert.NoError(t, viper.UnmarshalKey("prune", &pc2))
		assert.Equal(t, 2000, pc2.Threshold.Log.MaxLogs)
	}

	var pc pruneConfig
	assert.NoError(t, UnmarshalKey("prune", &pc))

	{ // test ViperSub(...) works with env var for type int
		assert.Equal(t, 1000, pc.Threshold.Log.MaxLogs)

		// supports to change env at runtime
		os.Setenv("CFX_PRUNE_THRESHOLD_LOG_MAXLOGS", "5000")
		var pc2 pruneConfig
		assert.NoError(t, UnmarshalKey("prune", &pc2))
		assert.Equal(t, 5000, pc2.Threshold.Log.MaxLogs)

		// multiple sub path
		os.Setenv("CFX_PRUNE_THRESHOLD_LOG_MAXLOGS", "15000")
		var llc logLimit
		assert.NoError(t, UnmarshalKey("prune.threshold.log", &llc))
		assert.Equal(t, 15000, llc.MaxLogs)
	}
}

func TestEnvWithoutConfig(t *testing.T) {
	reset()

	os.Setenv("CFX_TESTCONFIG_ENABLED", "true")
	os.Setenv("CFX_TESTCONFIG_SUBCONFIG_ENABLED", "true")

	var conf struct {
		Enabled   bool
		SubConfig struct {
			Enabled bool
		}
	}

	assert.NoError(t, UnmarshalKey("testConfig", &conf))
	assert.True(t, conf.Enabled)
	assert.True(t, conf.SubConfig.Enabled)
}

func TestEnvConfigMixed(t *testing.T) {
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

	// global config
	var config TestConfig
	assert.NoError(t, Unmarshal(&config))
	assert.Equal(t, ValConfig{"val1", "env2", "yml3", "env4"}, config.Foo.Bar)

	// sub config
	var sub ValConfig
	assert.NoError(t, UnmarshalKey("foo.bar", &sub))
	assert.Equal(t, ValConfig{"val1", "env2", "yml3", "env4"}, sub)
}

/*
 * For environment value, `UnmarshalKey` and `GetStringSlice` require different format.
 * - `UnmarshalKey` requires comma separated value, e.g. "a,b,c"
 * - `GetStringSlice` requires space separated value, e.g. "a b c"
 */
func TestEnvStringSlice(t *testing.T) {
	reset()

	var conf struct {
		Names1 []string
		Names2 []string
	}

	os.Setenv("CFX_TESTCONFIG_NAMES1", "a,b,c")
	os.Setenv("CFX_TESTCONFIG_NAMES2", "a b c")

	assert.NoError(t, UnmarshalKey("testConfig", &conf))
	assert.Equal(t, []string{"a", "b", "c"}, conf.Names1)
	assert.Equal(t, []string{"a b c"}, conf.Names2)

	assert.Equal(t, []string{"a,b,c"}, viper.GetStringSlice("testConfig.names1"))
	assert.Equal(t, []string{"a", "b", "c"}, viper.GetStringSlice("testConfig.names2"))
}

func TestUnmarshalEnvDateTypes(t *testing.T) {
	reset()

	type DataTypes struct {
		Int          int
		Bool         bool
		String       string
		Duration     time.Duration
		StringSlice  []string
		Map          map[int]ValConfig
		Address      testutil.HexAddress // Custom type implements the encoding.TextUnmarshaler interface
		AddressSlice []testutil.HexAddress
	}

	// init env
	os.Setenv("CFX_FOO_BAR_INT", "777")
	os.Setenv("CFX_FOO_BAR_BOOL", "true")
	os.Setenv("CFX_FOO_BAR_STRING", "teststring")
	os.Setenv("CFX_FOO_BAR_DURATION", "30m")
	os.Setenv("CFX_FOO_BAR_STRINGSLICE", "a,b,c")
	os.Setenv("CFX_FOO_BAR_MAP_1030_VAL1", "vvv1")
	os.Setenv("CFX_FOO_BAR_MAP_1030_VAL2", "vvv2")
	os.Setenv("CFX_FOO_BAR_MAP_1_VAL3", "vvv3")
	os.Setenv("CFX_FOO_BAR_MAP_1_VAL4", "vvv4")
	os.Setenv("CFX_FOO_BAR_ADDRESS", "0x86E7e8a956c781cc7385cBc29fdE0e737dE48b73")
	os.Setenv("CFX_FOO_BAR_ADDRESSSLICE", "0xdde37114971423be2497D190346AE57d82c5EbB7,0xE3bd412550FA07F1A3eBD1Fab285616016C38b96")

	expected := DataTypes{
		Int:         777,
		Bool:        true,
		String:      "teststring",
		Duration:    30 * time.Minute,
		StringSlice: []string{"a", "b", "c"},
		Map: map[int]ValConfig{
			1030: {"vvv1", "vvv2", "", ""},
			1:    {"", "", "vvv3", "vvv4"},
		},
		Address: testutil.MustParseHexAddress("0x86E7e8a956c781cc7385cBc29fdE0e737dE48b73"),
		AddressSlice: []testutil.HexAddress{
			testutil.MustParseHexAddress("0xdde37114971423be2497D190346AE57d82c5EbB7"),
			testutil.MustParseHexAddress("0xE3bd412550FA07F1A3eBD1Fab285616016C38b96"),
		},
	}

	// global config
	var config struct {
		Foo struct {
			Bar DataTypes
		}
	}
	assert.NoError(t, Unmarshal(&config))
	assert.Equal(t, expected, config.Foo.Bar)

	// sub config
	var sub DataTypes
	assert.NoError(t, UnmarshalKey("foo.bar", &sub))
	assert.Equal(t, expected, sub)
}
