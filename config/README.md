# Configuration Utilities

This package aims to initialize all common used modules at the entry point of program, including [viper](../viper/README.md), [log](../log/README.md), [metrics](../metrics/README.md), [alert](../alert/README.md) and [pprof](../pprof).

Generally, all configurations will be initialized at the entry point of program. Take `cobra` commands for example:

```go
import (
	"github.com/Conflux-Chain/go-conflux-util/config"
	"github.com/spf13/cobra"
)

func init() {
	cobra.OnInitialize(func() {
		config.MustInit("FOO")
	})
}

```

You could follow the example [config.yaml](./config.yaml) to setup your own configuration file. Generally, you could only overwrite configurations if the default value not suitable.