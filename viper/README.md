# Viper Utilities

This package provides utilities to initialize [viper](https://github.com/spf13/viper) in common, and fixes some bugs as well.

## Initialize Viper

Please use below method to initialize `viper` in common:

```go
func MustInit(envPrefix string, configPath ...string)
```

The parameter `envPrefix` is used to overwrite configurations via environment variables, e.g.

```shell
# evnPrefix is FOOAPP
export FOOAPP_API_ENDPOINT="http://localhost:12345"
```

The optional parameter `configPath` indicates the configuration file path to load (e.g. `./config-prod.yaml`). If not specified, the program will search for `config.xxx` (e.g. `config.yaml`, `config.json` or `config.toml`) file under current folder and `./config` folder.

As a best practice, we recommend to initialize `viper` via [config](../config/README.md).

## Bugfix

There is bug when unmarshaling configurations that overwritten by environment variables, please use below method instead.

```go
// E.g. load `foo` config from file.
var config FooConfig
viper.MustUnmarshalKey("foo", &config)
```
