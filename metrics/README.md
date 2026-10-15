# Metrics Utilities

This package enhances the original [go-metrics](https://github.com/rcrowley/go-metrics), and adds more helpful metric types.

## Initialize

It is highly recommended to initialize metrics via [config](../config/README.md). However, you could still initialize metrics programmatically.

Call `Init` or `MustInitFromViper` only once during application startup, before creating or using metrics.
An enabled InfluxDB reporter runs for the lifetime of the process and has no stop mechanism.
Reinitialization and dynamic disabling are unsupported: initializing again with InfluxDB enabled starts
another reporter, while calling `Init` with `Enabled: false` does not stop an existing reporter.
When using `report.InfluxDB` directly, it blocks after successful client initialization and is typically
run in a goroutine.

Programmatic initialization does not apply `default` tags. Set the reporting interval and HTTP timeout
explicitly, as shown below. `Timeout` applies to both Ping and write requests; a non-positive value
disables the HTTP timeout. Configuration decoding through this repository's Viper utility applies the
`10s` default timeout.

```go
package main

import (
	"time"

	"github.com/Conflux-Chain/go-conflux-util/metrics"
	"github.com/Conflux-Chain/go-conflux-util/metrics/report"
)

func main() {
	metrics.Init(metrics.Config{
		Enabled:        true,
		ReportInterval: 10 * time.Second,
		InfluxDb: &report.InfluxDBConfig{
			Host:     "http://localhost:8086",
			DB:       "metrics_db",
			Username: "user",
			Password: "pass",
			Timeout:  10 * time.Second,
		},
	})
}

```

## Percentage Metric Type

The `Percentage` interface records samples with `Mark` and returns their marked percentage with `Value`.
With samples, the value ranges from 0 to 100; for example, 99.38 means 99.38%.
Without samples, including when all samples in a time window have expired, `Value` returns `NaN`.
When metrics are disabled, constructors return a no-op implementation whose value is always 0.

Two constructors are available:

- `NewPercentage`: records overall statistics.
- `NewTimeWindowPercentage`: records statistics within a time window.

The registration helpers wrap `Percentage` to implement `go-metrics.GaugeFloat64`.
When a registered percentage returns `NaN`, InfluxDB reporting skips the measurement,
and `GetMetrics` returns a `nil` value that is encoded as JSON `null`.

See examples below:

```go
package main

import "github.com/Conflux-Chain/go-conflux-util/metrics"

func main() {
	overallPercentage := metrics.GetOrRegisterPercentage("business.overall.success.rate")
	recentPercentage := metrics.GetOrRegisterTimeWindowPercentageDefault("business.recent.success.rate")
	
	err := doBusinessLogic()
	
	overallPercentage.Mark(err == nil)
	recentPercentage.Mark(err == nil)
}

```

## Report for Custom Metric

Any custom metric type could implements the [Reportable](./report/reportable.go) interface, so as to report metric values to influxdb.

```go
type Reportable interface {
	Report() (metricType string, fields map[string]interface{})
}
```

## JSON RPC

Program could export all metrics based on JSON RPC, there is a default implementation available [here](./api.go).
