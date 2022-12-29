# Golang Development Utilities
Utilities for golang developments on Conflux blockchain, especially for backend service.

|Module|Description|
|------|-------|
|[Alert](#alert)|Send notification message to dingtalk.|
|[API](#api)|REST API utilities based on [gin](https://github.com/gin-gonic/gin).|
|[Config](#config)|Initialize all modules.|
|[HTTP](#http)|Provides common used middlewares.|
|[Log](#log)|Based on [logrus](https://github.com/sirupsen/logrus) and integrated with [Alert](#alert).|
|[Metrics](#metrics)|To monitor system runtime.|
|[Rate Limit](#rate-limit)|Utilities to limit request rate.|
|[Store](#store)|Provides utilities to initialize database.|
|[Viper](#viper)|To fix some issues of original [viper](https://github.com/spf13/viper).|

## Alert
Before sending any message to dingtalk, client should initialize dingtalk robot when setup program.

Initialize programmatically:
```go
alert.InitDingTalk(config *DingTalkConfig, tags []string)
alert.SendDingTalkTextMessage(level, brief, detail string)
```

Or, initialize from configuration file, which is recommended:
```go
alert.MustInitFromViper()
```

Moreover, alert could be integrated with `log` module, so as to send messages to dingtalk when `warning` or `error` logs occurred.

## API
This module provides common HTTP responses along with standard errors in JSON format.

Uniform JSON message in HTTP response body:
```go
type BusinessError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}
```

`Code` **0** indicates success, and `Data` is an object indicates the return value of REST API. `Code` with non-zero value indicates any error occurred, please refer to the `Message` and `Data` fields for more details. There are some pre-defined errors as below:

- 1: Invalid parameter, see `Data` for detailed error.
- 2: Internal server error, see `Data` for detailed error.
- 3: Too many requests, see `Data` for detailed error.

To distinguish backend service error and gateway error, we only use `200` and `600` as HTTP response status code:

- 200: success, or known business error, e.g. entity not found.
- 600: unexpected system error, e.g. database error, io error.

We recommend to initialize REST API from configuration file. Client only requires to provide a factory to setup controllers.

```go
// Setup controllers.
factory := func(router *gin.Engine) {
    router.GET("url", api.Wrap(controller))
}

// Start REST API server in a separate goroutine.
go api.MustServeFromViper(factory)
```

## Config
Initialize all modules at the entry point of program, including [viper](#viper), [log](#log), [metrics](#metrics) and [alert](#alert).

```go
config.MustInit(viperEnvPrefix string)
```

The `viperEnvPrefix` is used to overwrite configurations from environment. E.g. if the `viperEnvPrefix` is `FOO`, then client could set environment as below to overwrite config `alert.dingTalk.secret`:

```sh
FOO_ALERT_DINGTALK_SECRET='dsafsadf'
```

You could follow the example `config.yaml` under config folder to setup your own configuration file. Generally, you could only overwrite configurations if the default value not suitable.

## HTTP
Provides utilities to hook middlewares to HTTP handler, e.g. remote address, API key and rate limit.

## Log
We recommend to initialize log module from configuration file, and allow to send dingtalk messages when `warning` or `error` messages occurred.

## Metrics
We recommend to initialize metrics module from configuration file. Client could also configure influxdb to report metrics periodically. See `MetricsConfig` for more details.

## Rate Limit
Provides basic rate limit algorithms, including fixed window, token bucket, along with utilities for HTTP middleware.

Note, rate limit middleware depends on the HTTP middleware `RealIP`.

## Store
We recommend to initialize store module from configuration file.

```go
config := mysql.MustNewConfigFromViper()
db := config.MustOpenOrCreate(tables ...interface{})
```

## Viper
Fixes issues when unmarshal configurations from environment. A simple way to load configuration from file is as below:

```go
viper.MustUnmarshalKey(key string, valPtr interface{}, resolver ...ValueResolver)

// E.g. load `foo` config from file.
var config FooConfig
viper.MustUnmarshalKey("foo", &config)
```
