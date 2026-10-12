package viper

import (
	"os"
	"reflect"
	"sort"
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

// ValueResolver optionally overrides the value of a full configuration key,
// before any subtree prefix is removed. Returning false falls back to viper.Get;
// returning (nil, true) skips writing that key.
type ValueResolver func(key string) (interface{}, bool)

// allEnv collects environment variables matching envKeyPrefix, removes that
// prefix, and converts their names to lowercase dot-separated configuration keys.
// Empty values are retained for key discovery; viper.Get applies AllowEmptyEnv
// and configuration precedence when resolving each value.
func allEnv() map[string]string {
	result := make(map[string]string)

	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, envKeyPrefix) {
			continue
		}

		key, value, ok := strings.Cut(v, "=")
		if !ok {
			continue
		}

		key = strings.TrimPrefix(key, envKeyPrefix)
		key = strings.ReplaceAll(key, "_", ".")
		key = strings.ToLower(key)

		result[key] = value
	}

	return result
}

// keys returns sorted, unique keys known to Viper or discovered in environment
// variables that match the specified prefix. Parent keys precede their children.
//
// An empty prefix includes all keys.
func keys(prefix string) []string {
	prefix = strings.ToLower(prefix)
	result := make(map[string]bool)

	// Load keys known to Viper (including files, overrides and bindings).
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

	orderedKeys := make([]string, 0, len(result))
	for key := range result {
		orderedKeys = append(orderedKeys, key)
	}

	// Write parents before children because a parent map may be only a partial
	// value from a higher-priority source. Writing it after its children could
	// discard independently resolved child values.
	sort.Strings(orderedKeys)

	return orderedKeys
}

// sub builds a new Viper with resolved settings under name. An empty name includes
// all keys; otherwise only descendants of name are included and the "name."
// prefix is removed from their keys.
//
// Keys include environment-only settings and are processed parent-first. Values
// come from viper.Get unless the first optional resolver handles the full key.
// Nil values are not written. Resolving each key separately preserves environment
// overrides that Viper's native subtree unmarshalling can miss.
// See https://github.com/spf13/viper/issues/1012#issuecomment-757862260.
func sub(name string, resolver ...ValueResolver) *viper.Viper {
	subViper := viper.New()

	if len(name) > 0 {
		name += "."
	}

	for _, key := range keys(name) {
		subKey := key[len(name):]

		var value interface{}
		if len(resolver) == 0 {
			value = viper.Get(key)
		} else if val, ok := resolver[0](key); ok {
			value = val
		} else {
			value = viper.Get(key)
		}

		// Viper treats a higher-priority scalar parent as shadowing lower-priority
		// descendants. For example, CFX_FOO makes Get("foo.name") return nil even
		// when foo.name exists in the config file. Writing that nil child would
		// turn the scalar parent into a map and discard its resolved value.
		// Non-nil children are still written because a higher-priority child source
		// may legitimately override a lower-priority scalar parent.
		if value != nil {
			subViper.Set(subKey, value)
		}
	}

	return subViper
}

// MustUnmarshalKey calls UnmarshalKey with the same key and resolver semantics.
//
// If UnmarshalKey returns an error, it logs the error with logrus.Fatal and exits
// the process. This error path does not use panic and cannot be caught by recover.
func MustUnmarshalKey(key string, valPtr interface{}, resolver ...ValueResolver) {
	if err := UnmarshalKey(key, valPtr, resolver...); err != nil {
		logrus.WithError(err).
			WithField("valType", reflect.TypeOf(valPtr)).
			Fatal("Failed to unmarshal data from viper")
	}
}

// UnmarshalKey decodes the configuration subtree at key into valPtr.
// An empty key decodes the entire configuration, including environment-only keys.
// valPtr must be a non-nil pointer to a struct.
//
// Only the first optional resolver is used; it receives full configuration keys.
// Without a resolver override, values follow viper.Get's precedence and empty-env
// policy. The default decode hook supports comma-separated string slices; a
// resolver can select another getter or value format when needed.
//
// Default tags are applied after decoding: explicit zero values such as false,
// 0, and "" cannot override nonzero defaults on fields supported by go-defaults.
func UnmarshalKey(key string, valPtr interface{}, resolver ...ValueResolver) error {
	subViper := sub(key, resolver...)
	if err := subViper.Unmarshal(valPtr, viper.DecodeHook(defaultDecodeHook)); err != nil {
		return err
	}

	// Apply defaults after decoding to avoid merging configured slices with
	// prefilled defaults (e.g. [configured] must not become [configured, two]).
	defaults.SetDefaults(valPtr)

	return nil
}

// Unmarshal decodes the entire configuration, including environment-only keys,
// into valPtr, which must be a non-nil pointer to a struct. Values follow
// viper.Get's precedence and empty-env policy. Decoder options are applied after
// the default decode hook, allowing callers to override it.
//
// Default tags are applied after decoding: explicit zero values such as false,
// 0, and "" cannot override nonzero defaults on fields supported by go-defaults.
func Unmarshal(valPtr any, opts ...viper.DecoderConfigOption) error {
	subViper := sub("")

	opts = append([]viper.DecoderConfigOption{viper.DecodeHook(defaultDecodeHook)}, opts...)

	if err := subViper.Unmarshal(valPtr, opts...); err != nil {
		return err
	}

	// Keep the same post-decode ordering as UnmarshalKey so configured slices
	// do not retain elements from prefilled default slices.
	defaults.SetDefaults(valPtr)

	return nil
}
