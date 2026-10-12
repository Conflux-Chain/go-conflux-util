package viper

import (
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalPrecedence(t *testing.T) {
	tests := []struct {
		name       string
		env        string
		override   string
		flag       string
		allowEmpty bool
		want       string
	}{
		{name: "environment beats file and unchanged flag", env: "environment", want: "environment"},
		{name: "Set beats flag and environment", env: "environment", override: "explicit", flag: "flag", want: "explicit"},
		{name: "changed flag beats environment", env: "environment", flag: "flag", want: "flag"},
		{name: "empty environment falls back to file", env: "", want: "file"},
		{name: "empty environment allowed explicitly", env: "", allowEmpty: true, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reset(t)
			viper.SetConfigType("json")
			require.NoError(t, viper.ReadConfig(strings.NewReader(`{"foo":{"bar":{"name":"file","fileOnly":"retained"}}}`)))
			t.Setenv("CFX_FOO_BAR_NAME", tt.env)
			t.Setenv("CFX_FOO_BAR_ENVONLY", "env-only")
			viper.AllowEmptyEnv(tt.allowEmpty)

			flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
			flags.String("name", "flag-default", "")
			if tt.flag != "" {
				require.NoError(t, flags.Set("name", tt.flag))
			}
			require.NoError(t, viper.BindPFlag("foo.bar.name", flags.Lookup("name")))
			if tt.override != "" {
				viper.Set("foo.bar.name", tt.override)
			}

			var config struct {
				Foo struct {
					Bar struct{ Name, FileOnly, EnvOnly string }
				}
			}
			require.NoError(t, Unmarshal(&config))
			assert.Equal(t, tt.want, config.Foo.Bar.Name)
			assert.Equal(t, "retained", config.Foo.Bar.FileOnly)
			assert.Equal(t, "env-only", config.Foo.Bar.EnvOnly)
		})
	}
}

func TestUnmarshalParentOverrideShadowsEnvironmentChild(t *testing.T) {
	for _, method := range []string{"Unmarshal", "UnmarshalKey"} {
		t.Run(method, func(t *testing.T) {
			reset(t)
			viper.Set("foo.name", "explicit")
			t.Setenv("CFX_FOO_NAME_CHILD", "environment")

			var config struct{ Foo struct{ Name string } }
			if method == "Unmarshal" {
				require.NoError(t, Unmarshal(&config))
			} else {
				require.NoError(t, UnmarshalKey("foo", &config.Foo))
			}
			assert.Equal(t, "explicit", config.Foo.Name)
		})
	}
}

func TestUnmarshalEnvironmentScalarParentShadowsFileChild(t *testing.T) {
	reset(t)
	viper.SetConfigType("json")
	require.NoError(t, viper.ReadConfig(strings.NewReader(`{"foo":{"name":"file"}}`)))
	t.Setenv("CFX_FOO", "environment")

	var config struct{ Foo string }
	require.NoError(t, Unmarshal(&config))
	assert.Equal(t, "environment", config.Foo)
}

func TestUnmarshalEnvironmentParentPreservesResolvedChildren(t *testing.T) {
	reset(t)
	viper.SetConfigType("json")
	require.NoError(t, viper.ReadConfig(strings.NewReader(`{"foo":{"name":"file","sibling":"retained"}}`)))
	viper.Set("foo.name", "explicit")
	// Get("foo") returns a partial override map. It must not overwrite the
	// independently resolved child "foo.sibling" from the config file.
	t.Setenv("CFX_FOO", "")
	t.Setenv("CFX_FOO_NAME", "environment")

	var config struct {
		Foo struct{ Name, Sibling string }
	}
	require.NoError(t, Unmarshal(&config))
	assert.Equal(t, "explicit", config.Foo.Name)
	assert.Equal(t, "retained", config.Foo.Sibling)
}

func TestUnmarshalKeyEnvironmentParentPreservesResolvedChildren(t *testing.T) {
	for _, parentEnv := range []string{"", "parent-value"} {
		t.Run("parent="+parentEnv, func(t *testing.T) {
			reset(t)
			viper.SetConfigType("json")
			require.NoError(t, viper.ReadConfig(strings.NewReader(`{"foo":{"bar":{"name":"file","sibling":"retained"}}}`)))
			viper.Set("foo.bar.name", "explicit")
			t.Setenv("CFX_FOO_BAR", parentEnv)

			wantSibling := "retained"
			if parentEnv != "" {
				// A changed flag resolves before the nonempty parent environment
				// value, but is absent from Get("foo.bar")'s partial override map.
				flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
				flags.String("sibling", "", "")
				require.NoError(t, flags.Set("sibling", "flag"))
				require.NoError(t, viper.BindPFlag("foo.bar.sibling", flags.Lookup("sibling")))
				wantSibling = "flag"
			}

			// Repeated decoding exposes the previously unspecified map order.
			for i := 0; i < 32; i++ {
				var config struct {
					Bar struct{ Name, Sibling string }
				}
				require.NoError(t, UnmarshalKey("foo", &config))
				require.Equal(t, "explicit", config.Bar.Name)
				require.Equal(t, wantSibling, config.Bar.Sibling)
			}
		})
	}
}
