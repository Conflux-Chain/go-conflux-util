package viper

import (
	"os"
	"reflect"
	"strings"

	"github.com/mcuadros/go-defaults"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// ValueResolver defines custom method to get value by key from viper settings.
type ValueResolver func(key string) (interface{}, bool)

// keys load all viper keys both from config file and env vars.
func keys(prefix string) map[string]bool {
	result := make(map[string]bool)

	// load keys from configuration file
	for _, v := range viper.AllKeys() {
		if strings.HasPrefix(v, prefix) {
			result[v] = true
		}
	}

	// load keys from environments
	prefix = strings.ToLower(prefix)
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, envKeyPrefix) {
			continue
		}

		fields := strings.SplitN(v, "=", 2)
		key := strings.TrimPrefix(fields[0], envKeyPrefix)
		key = strings.ReplaceAll(key, "_", ".")
		key = strings.ToLower(key)

		if strings.HasPrefix(key, prefix) {
			result[key] = true
		}
	}

	return result
}

// viperSub is used to fix viper.Sub not respect enviroment variables while unmarshal into struct.
//
// More info for this fix: https://github.com/spf13/viper/issues/1012#issuecomment-757862260
// Besides, for enviroment variables set to be used to override some special types like []string,
// specified key-getter method must be provided to load the envoriment variables correctly.
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
	defaults.SetDefaults(valPtr)
	subViper := sub(key, resolver...)
	return subViper.Unmarshal(valPtr)
}
