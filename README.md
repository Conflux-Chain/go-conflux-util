# Golang Development Utilities
Utilities for golang developments on Conflux blockchain, especially for backend service.

|Module|Comment|
|------|-------|
|[Alert](#alert)|Send notification message to dingtalk.|
|[API](#api)|REST API utilities based on [gin](https://github.com/gin-gonic/gin).|
|[Config](#config)|Initialize all modules.|
|[Log](#log)|Based on [logrus](https://github.com/sirupsen/logrus) and integrated with [Alert](#alert).|
|[Metrics](#metrics)|To monitor system runtime.|
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

To distinguish backend service error and gateway error, we only use `200` and `600` as HTTP response status code:

- 200: success, or known business error, e.g. entity node found.
- 600: unexpected system error, e.g. database error, io error.

We recommend to initialize REST API from configuration file. Client only requires to provide a factory to setup controllers.

```go
factory := func(router *gin.Engine) {
    router.GET("url", api.Wrap(controller))
}

api.MustServeFromViper(factory)
```

## Config
Initialize all modules at the entry point of program.

```go
config.MustInit(viperEnvPrefix string)
```

The `viperEnvPrefix` is used for overwrite configurations from environment. E.g. if the `viperEnvPrefix` is `FOO`, then client could set environment as below to overwrite config `alert.dingTalk.secret`:

```
FOO_ALERT_DINGTALK_SECRET='dsafsadf'
```

## Log
We recommend to initialize log module from configuration file, and allow to send dingtalk messages when `warning` or `error` messages occurred.

## Metrics
We recommend to initialize metrics module from configuration file. Client could also configure influxdb to report metrics periodically. See `MetricsConfig` for more details.

## Store
We recommend to initialize store module from configuration file.

```go
config := mysql.MustNewConfigFromViper()
db := config.MustOpenOrCreate(tables ...interface{})
```

## Viper
Fixes issues when unmarshal configurations from environment.
