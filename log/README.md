# Log Utilities

This package enhances the original [logrus](https://github.com/sirupsen/logrus), and supports to hook [alert](../alert/README.md).

## Configurations

Currently, there are several configurations available:

- Set log level.
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

## Hook with Alert

Developers can configure the alert hook to set up default notification channels for sending alert messages when `warning` or `error` logs occur. You can also customize notifications by specifying the target channel(s) through the `@channel` field in a Logrus entry.

```go
// Send alert to the 'tgrobot' channel only.
logrus.WithField("@channel": "tgrobot").Warn("Some warning occurred")
```