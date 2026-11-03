# Configuration Utilities

This package aims to initialize all common used modules at the entry point of program, including [viper](../viper/README.md), [log](../log/README.md), [metrics](../metrics/README.md), [alert](../alert/README.md) and [pprof](../pprof).

Generally, all configurations will be initialized at the entry point of program. Take `cobra` commands for example:

Choose exactly one `MustInit*` entry point and call it once during process startup,
before using the initialized components concurrently. Repeated initialization and
hot reload are unsupported: they can duplicate alert hooks or background reporters,
or fail to bind an already occupied profiling endpoint.

```go
import (
	"github.com/Conflux-Chain/go-conflux-util/config"
	"github.com/spf13/cobra"
)

func init() {
	// Init config with default env prefix "APP_".
	cobra.OnInitialize(config.MustInitDefault)

	// Alternatively, replace the registration above with:
	// cobra.OnInitialize(func() {
	// 	config.MustInit("FOO") // Uses the "FOO_" prefix.
	// })
}

```

You could follow the [config.yaml.example](./config.yaml.example) or [.env.example](.env.example) to setup your own configuration file. Generally, you could only overwrite configurations if the default value not suitable.

Replace the example credentials and addresses before use. YAML values such as
`${TOKEN}` are literal strings; this package does not expand environment-variable
placeholders inside YAML. Instead, use the configuration's environment keys. For
example, with `MustInitDefault`, `APP_ALERT_CHANNELS_DINGROBOT_SECRET` overrides
`alert.channels.dingrobot.secret`. With `MustInit("FOO")`, use the `FOO_` prefix.

## Shutdown scope

`MustInitWithCtx` connects only configured asynchronous log-alert workers to the
provided context and wait group. Both arguments must be non-nil; otherwise,
logging is initialized without cancellation handling. Metrics reporting and pprof
continue to run for the lifetime of the process.

After initialization returns, stop log producers, cancel the context, and then
call `wg.Wait()`. The log hook registers its own work with `wg.Add`; do not add an
extra count for this initialization. Waiting covers those workers and their
best-effort queue drain, not every initialized service. `StopTimeout` is a queue
draining budget after workers exit, not a bound on the total shutdown time.

## Initialization failures

Configuration loading/decoding, alert/log initialization, and pprof listening
errors are logged with `Fatal`. With the default Logrus exit function, this exits
the process without running Go defers; `recover` cannot intercept it. Invalid
component configuration may instead panic, such as a nonpositive metrics report
interval when reporting is enabled.

Some components start asynchronously. For example, an InfluxDB client creation
failure is logged and stops the reporter without exiting the process. Returning
from initialization does not guarantee that every background component is ready.
