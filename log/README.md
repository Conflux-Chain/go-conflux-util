# Log Utilities

This package enhances the original [logrus](https://github.com/sirupsen/logrus), and supports to hook [alert](../alert/README.md).

## Configurations

Currently, there are several configurations available:

- Set default log level (default value is `info`).
- Set log levels for specific modules.
- Force or disable colors in Console or output file.
- Hook with `alert`.
- Output to file with rotation.

## Initialize

We recommend to initialize the log module from a configuration file or environment variables.

```go
// Initialize logging by specifying configurations
log.MustInit(conf)

// or Initialize logging from configurations loaded by viper
log.MustInitFromViper()
```

## Log with Module

Application could specify different log levels for different moduels. Please use pre-defined factory method to create a logger with specific module name and log level.

```go
log.WithModule("rpc").Info("Info message")
log.WithModule("rpc").Debug("Debug message")
```

## Hook with Alert

Developers can configure the alert hook to set up default notification channels for sending alert messages when `warning` or `error` logs occur. You can also customize notifications by specifying the target channel(s) through the `@channel` field in a Logrus entry.

```go
// Send alert to the 'tgrobot' channel only.
logrus.WithField("@channel", "tgrobot").Warn("Some warning occurred")
```