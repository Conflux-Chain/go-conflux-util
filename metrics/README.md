# Metrics Utilities

This package enhances the original [go-metrics](https://github.com/rcrowley/go-metrics), and adds more helpful metric types.

## Initialize

It is highly recommend to initialize metrics via [config](../config/README.md). However, you could still initialize metrics programmatically.

```go
metrics.Init(config Config)
```
## Percentage Metric Type

The `Percentage` metric type is used for percentage statistic, which ranges in [0,100]. E.g. 99.38 means 99.38%. There are 2 implementations available:

- `StandardPercentage`: overall statistics.
- `TimeWindowPercentage`: statistics in the past period of time based on time window.

## Report for Custom Metric

Any custom metric type could implements the [Reportable](./report/reportable.go) interface, so as to report metric values to influxdb.

## JSON RPC

Program could export all metrics based on JSON RPC, there is a default implementation available [here](./api.go).
