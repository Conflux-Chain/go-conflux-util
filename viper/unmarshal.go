package viper

import (
	"os"
	"reflect"
	"strings"

	"github.com/mcuadros/go-defaults"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var defaultDecodeHook = mapstructure.ComposeDecodeHookFunc(
	mapstructure.StringToTimeDurationHookFunc(),
	mapstructure.StringToSliceHookFunc(","),
	mapstructure.TextUnmarshallerHookFunc(),
)

// ValueResolver defines custom method to get value by key from viper settings.
type ValueResolver func(key string) (interface{}, bool)

// allEnv loads all environment variables of specified `envKeyPrefix` in kv format.
func allEnv() map[string]string {
	result := make(map[string]string)

	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, envKeyPrefix) {
			continue
		}

		fields := strings.SplitN(v, "=", 2)

		key := strings.TrimPrefix(fields[0], envKeyPrefix)
		key = strings.ReplaceAll(key, "_", ".")
		key = strings.ToLower(key)

		result[key] = fields[1]
	}

	return result
}

// keys load all viper keys both from config file and env vars.
func keys(prefix string) map[string]bool {
	prefix = strings.ToLower(prefix)
	result := make(map[string]bool)

	// load keys from configuration file
	for _, v := range viper.AllKeys() {
		if strings.HasPrefix(v, prefix) {
			result[v] = true
		}
	}

	// load keys from environments
	for key := range allEnv() {
		if strings.HasPrefix(key, prefix) {
			result[key] = true
		}
	}

	return result
}

// sub is used to fix viper.Sub not respect environment variables while unmarshal into struct.
//
// More info for this fix: https://github.com/spf13/viper/issues/1012#issuecomment-757862260
// Besides, for environment variables set to be used to override some special types like []string,
// specified key-getter method must be provided to load the environment variables correctly.
func sub(name string, resolver ...ValueResolver) *viper.Viper {
	subViper := viper.New()

	for key := range keys(name + ".") {
		subKey := key[len(name)+1:]

		var value interface{}
		if len(resolver) == 0 {
			value = viper.Get(key)
		} else if val, ok := resolver[0](key); ok {
			value = val
		} else {
			value = viper.Get(key)
		}

		subViper.Set(subKey, value)
	}

	return subViper
}

// MustUnmarshalKey unmarshal setting to value pointer by key from viper.
//
// Note that it will panic and exit if failed to unmarshal value from key.
func MustUnmarshalKey(key string, valPtr interface{}, resolver ...ValueResolver) {
	if err := UnmarshalKey(key, valPtr, resolver...); err != nil {
		logrus.WithError(err).
			WithField("valType", reflect.TypeOf(valPtr)).
			Fatal("Failed to unmarshal data from viper")
	}
}

// UnmarshalKey unmarshal setting to value pointer by key from viper.
//
// Provide custom value resolver if unmarshalling some special types like slice.
// Note that valPtr must be some value pointer and not be nil.
func UnmarshalKey(key string, valPtr interface{}, resolver ...ValueResolver) error {
	subViper := sub(key, resolver...)
	if err := subViper.Unmarshal(valPtr, viper.DecodeHook(defaultDecodeHook)); err != nil {
		return err
	}

	defaults.SetDefaults(valPtr)

	return nil
}

// Unmarshal unmarshals the config into a Struct.
func Unmarshal(valPtr any, opts ...viper.DecoderConfigOption) error {
	copy := viper.New()

	for k, v := range viper.AllSettings() {
		copy.Set(k, v)
	}

	for k, v := range allEnv() {
		copy.Set(k, v)
	}

	opts = append([]viper.DecoderConfigOption{viper.DecodeHook(defaultDecodeHook)}, opts...)

	if err := copy.Unmarshal(valPtr, opts...); err != nil {
		return err
	}

	defaults.SetDefaults(valPtr)

	return nil
}
