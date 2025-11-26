# Viper Utilities

This package provides utilities to initialize [viper](https://github.com/spf13/viper) in common, and fixes some bugs as well.

## Initialize Viper

Please use below method to initialize `viper` in common:

```go
package main

import "github.com/Conflux-Chain/go-conflux-util/viper"

func main() {
	// initialize viper with environment variable prefix "FOO"
	viper.MustInit("FOO")

	// initialize viper with environment variable prefix "FOO" and config file "config-prod.yaml"
	viper.MustInit("FOO", "config-prod.yaml")
}
```

The 1st parameter `envPrefix` is used to overwrite configurations via environment variables, e.g.

```shell
# evnPrefix is FOO
export FOO_API_ENDPOINT="http://localhost:12345"
```

The 2nd optional parameter `configPath` indicates the configuration file path to load (e.g. `./config-prod.yaml`). If not specified, the program will search for `config.xxx` (e.g. `config.yaml`, `config.json` or `config.toml`) file under current folder and `./config` folder.

As a best practice, we recommend to initialize `viper` via [config](../config/README.md).

## Bugfix

There is bug when unmarshaling configurations that overwritten by environment variables, please use below method instead.

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
