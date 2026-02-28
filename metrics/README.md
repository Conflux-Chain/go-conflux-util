# Metrics Utilities

This package enhances the original [go-metrics](https://github.com/rcrowley/go-metrics), and adds more helpful metric types.

## Initialize

It is highly recommended to initialize metrics via [config](../config/README.md). However, you could still initialize metrics programmatically.

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
		},
	})
}

```
## Percentage Metric Type

The `Percentage` metric type is used for percentage statistic, which ranges in [0,100]. E.g. 99.38 means 99.38%. There are 2 implementations available:

- `StandardPercentage`: is used for overall statistics.
- `TimeWindowPercentage`: is used for statistics in the past period of time based on time window.

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
