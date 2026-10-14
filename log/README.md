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

Applications can configure independent module levels, for example `info,rpc=debug,db=error`. Initialize logging before creating module entries. Explicitly configured modules reuse their own logger; other modules use the global logger and its default level.

```go
log.WithModule("rpc").Info("Info message")
log.WithModule("rpc").Debug("Debug message")
```

Module loggers inherit the output target, formatter, hooks and other logger settings at initialization. Later changes to the global logger's level, output, formatter or hook registry do not update these module loggers. The output target, formatter and hook instances are shared, so custom implementations must support concurrent use across loggers. Complete initialization before starting concurrent logging.

## Hook with Alert

Developers can configure the alert hook to set up default notification channels for sending alert messages when `warning` or `error` logs occur. You can also customize notifications by specifying the target channel(s) through the `@channel` field in a Logrus entry.

```go
// Send alert to the 'tgrobot' channel only.
logrus.WithField("@channel", "tgrobot").Warn("Some warning occurred")
```

### Asynchronous alert shutdown

When asynchronous alerts are enabled, use `log.MustInitWithCtx` or `log.MustInitWithCtxFromViper` with a context and wait group to track shutdown. Direct hook users can use `hook.NewAsyncHookWithCtx`. Stop and wait for log producers before cancelling the hook context, then wait for shutdown to complete.

Shutdown waits for workers to exit before starting `StopTimeout` to drain the remaining queue. This is a best-effort draining budget, not a total shutdown deadline: it cannot interrupt an underlying `Hook.Fire` call or the final shutdown notification. The underlying hook must limit its own I/O duration; a blocked call can keep `wg.Wait()` blocked beyond `StopTimeout`.

Alert delivery is best effort. A full queue, drain timeout, or send failure can result in lost alerts; graceful shutdown does not guarantee delivery of every alert.
