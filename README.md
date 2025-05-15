# Golang Development Utilities
Utilities for golang developments on Conflux blockchain, especially for backend service.

|Module|Description|
|------|-------|
|[Alert](#alert)|Send notification messages to DingTalk, Telegram, SMTP email or PagerDuty.|
|[API](#api)|REST API utilities based on [gin](https://github.com/gin-gonic/gin).|
|[Config](#config)|Initialize all modules.|
|[DLock](#distributed-lock)|Utilities for distributed lock.|
|[Health](#health)|Utilities for health management.|
|[HTTP](#http)|Provides common used middlewares.|
|[Log](#log)|Based on [logrus](https://github.com/sirupsen/logrus) and integrated with [Alert](#alert).|
|[Metrics](#metrics)|To monitor system runtime.|
|[Rate Limit](#rate-limit)|Utilities to limit request rate.|
|[Store](#store)|Provides utilities to initialize database.|
|[Viper](#viper)|To fix some issues of original [viper](https://github.com/spf13/viper).|

## Alert
Before sending any message to notification channels, the client should create a channel robot. To construct a channel robot programmatically:

```go
// Construct a notification channel imperatively
var notifyCh alert.Channel

// DingTalk Channel
notifyCh = alert.NewDingTalkChannel(...)
// or PagerDuty Channel
notifyCh = alert.NewPagerDutyChannel(...)
// or Smtp email Channel
notifyCh = alert.NewSmtpChannel(...)
// or Telegram Channel
notifyCh = alert.NewTelegramChannel(...)
```

Alternatively, you can initialize the alert channels from configuration file or environment variables, which is recommended.

```go
// Initialize the alert channels from configurations loaded by viper
alert.MustInitFromViper()
// After initialization, you can retrieve the notification channel using a unique channel ID
notifyCh := alert.DefaultManager().Channel(chID)
```

Once the channel is initialized, you can send a notification message through the channel:

```go
notifyCh.Send(context.Background(), &alert.Notification{
    Title:    "Alert testing",
    Severity: alert.SeverityLow,
    Content: `This is a test notification message`,
})
```

Moreover, alert can be integrated with [log](#log) module, so as to send alerting message when `warning` or `error` logs occurred.

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

You could follow the example [config.yaml](./config/config.yaml) under config folder to setup your own configuration file. Generally, you could only overwrite configurations if the default value not suitable.

## Distributed Lock
The distributed lock ensures atomicity in a distributed environment, such as leader election for achieving high availability.

To create a distributed lock, you need to specify a storage backend. We provide the `MySQLBackend` which handles reading and writing lock information in a MySQL table. Alternatively, you can implement your own storage backend using Redis, etcd, ZooKeeper, or other options.

```go
// Construct a lock manager with customized storage backend.
lockMan := dlock.NewLockManager(backend)
```

Alternatively, you can construct a lock manager with a convenient MySQL backend by using configuration files or environment variables.

```go
// Construct a lock manager with a MySQL backend from configurations loaded by viper
lockMan = dlock.NewLockManagerFromViper()
```

To acquire and release a lock, you can use:

```go
intent := NewLockIntent("dlock_key", 15 * time.Second)
// Acquire a lock with key name "dlock_key" for 15 seconds
lockMan.Acquire(context.Background(), intent)
// Release the lock immediately
lockMan.Release(context.Background(), intent)
```

## Health
Provides utilities for error tolerant health monitoring to avoid massive duplicated alerts.

Generally, system shall not report failure if auto resolved in a short time. However, system should report failure if not solved in a short time, and periodically remind failure if unrecovered for a long time.

- [Counter](./health/counter.go): manage health status based on the number of continous failures.
- [TimedCounter](./health/timed_counter.go): manage health status based on duration since the first failure.

## HTTP
Provides utilities to hook middlewares to HTTP handler, e.g. remote address, API key and rate limit.

## Log
We recommend initializing the log module from a configuration file or environment variables.

```go
// Initialize logging by specifying configurations
log.MustInit(conf)
// or Initialize logging from configurations loaded by viper
log.MustInitFromViper()
```

Additionally, you can configure the alert hook to set up default notification channels for sending alert messages when `warning` or `error` logs occur. You can also customize notifications by specifying the target channel(s) through the `@channel` field in a Logrus entry.

```go
// Send alert to the 'tgrobot' channel instead.
logrus.WithField("@channel": "tgrobot").Warn("Some warning occurred")
```

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
