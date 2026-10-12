# Viper Utilities

This package provides utilities to initialize [viper](https://github.com/spf13/viper) in common, and fixes some bugs as well.

## Initialize Viper

Please use below method to initialize `viper` in common:

```go
package main

import "github.com/Conflux-Chain/go-conflux-util/viper"

func main() {
	// initialize viper with environment variable prefix "FOO" and optional config.yaml file.
	viper.MustInit("FOO")

	// initialize viper with environment variable prefix "FOO" and custom config files:
	// config.yaml, config-prod.yaml.
	viper.MustInit("FOO", "config.yaml", "config-prod.yaml")
}
```

The 1st parameter `envPrefix` is used to overwrite configurations via environment variables. By default, the `.env` file will be auto loaded, which is common used in prod environment.

```shell
# Note, the 'export' keyword is optional in .env file.

# envPrefix is FOO
export FOO_API_ENDPOINT="http://localhost:12345"

# envPrefix is empty
export API_ENDPOINT="http://localhost:12345"
```

The 2nd optional parameter `configFiles` indicates the configuration files to load (e.g. `./config.yaml` and `./config-prod.yaml`). If not specified, the program will search for `config.xxx` (e.g. `config.yaml`, `config.json` or `config.toml`) file under current folder and `./config` folder.

As a best practice, we recommend to initialize `viper` via [config](../config/README.md).

## Bugfix

Viper's automatic environment lookup does not discover environment-only keys
during struct unmarshalling, and native subtree unmarshalling can miss nested
environment overrides. Use the following helpers to resolve these keys explicitly.

```go
// E.g. load `foo` config from file.
var fooConfig FooConfig
viper.MustUnmarshalKey("foo", &fooConfig)

// or load global config at a time.
var config Config
err := viper.Unmarshal(&config)
```

Note, above 2 methods support:

- Set default value based on `default` tag.
- Custom data type that implements the `encoding.TextUnmarshaler` interface.

`Unmarshal` and `UnmarshalKey` discover nested keys from environment variables even
when they are absent from the config file. By default, each value is resolved
through Viper: explicit `Set`
overrides and changed command-line flags take precedence over environment
variables, which still override config file values. Empty environment variables
are ignored unless `viper.AllowEmptyEnv(true)` is enabled on the underlying Viper.

`UnmarshalKey("foo", &config)` decodes descendants of `foo` after removing the
`foo.` prefix. An empty key, `UnmarshalKey("", &config)`, decodes the entire
configuration. Both decoding methods require a non-nil pointer to a struct.

`UnmarshalKey` accepts an optional `ValueResolver`; only the first resolver is
used. It receives full keys such as `foo.bar`, and returning `false` falls back to
`viper.Get`. Returning `(nil, true)` skips writing that key. Comma-separated string slices
are supported without a resolver; a resolver can supply another value format.
`MustUnmarshalKey` uses the same rules but logs a fatal error and exits the process
if decoding returns an error; that error path cannot be handled with `recover`.

Both methods apply `default` tags through `go-defaults` **after decoding**.
The library cannot distinguish an omitted value from an explicitly configured
zero value. On supported fields, values such as `false`, `0`, and `""` are therefore
replaced by a nonzero tagged default. Omit the corresponding `default` tag when
the configuration must allow these zero values.

This ordering also avoids merging configured slices with prefilled defaults.
For example, with `default:"[one,two]"`, a configured `["configured"]` remains
`["configured"]`, rather than becoming `["configured", "two"]`. An empty slice
may still receive a tagged default according to `go-defaults` semantics.
