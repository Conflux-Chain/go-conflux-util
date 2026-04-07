# Configuration Utilities

This package aims to initialize all common used modules at the entry point of program, including [viper](../viper/README.md), [log](../log/README.md), [metrics](../metrics/README.md), [alert](../alert/README.md) and [pprof](../pprof).

Generally, all configurations will be initialized at the entry point of program. Take `cobra` commands for example:

```go
import (
	"github.com/Conflux-Chain/go-conflux-util/config"
	"github.com/spf13/cobra"
)

func init() {
	// Init config with env prefix "FOO_"
	cobra.OnInitialize(func() {
		config.MustInit("FOO")
	})

	// Or init config with default env prefix "APP_"
	cobra.OnInitialize(config.MustInitDefault)
}

```

You could follow the [config.yaml.example](./config.yaml.example) or [.env.example](.env.example) to setup your own configuration file. Generally, you could only overwrite configurations if the default value not suitable.