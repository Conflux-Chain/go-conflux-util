package metrics

import (
	"fmt"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/metrics/report"
	"github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/rcrowley/go-metrics"
	"github.com/sirupsen/logrus"
)

const (
	// default exponentially-decaying metrics sample reservoir size
	expDecaySampleReservoirSize int = 1028

	// default exponentially-decaying metrics sample alpha
	expDecaySampleAlpha float64 = 0.015
)

// Config is metrics configurations such as influxdb settings.
type Config struct {
	// switch to turn on or off metrics
	Enabled bool
	// Interval to report metrics to InfluxDB; must be positive when reporting is enabled.
	// Direct calls to Init must set this explicitly; Init does not apply default tags.
	ReportInterval time.Duration `default:"10s"`
	// settings for influxdb to be reported to
	InfluxDb *report.InfluxDBConfig
}

// MustInitFromViper inits metrics from viper settings.
// Call it only once during application startup, before creating or using metrics.
// Like Init, it does not support reinitialization or dynamic disabling.
//
// Viper must be initialized first. Configuration decoding errors are logged with
// logrus.Fatal and terminate the process; recover cannot catch this error path.
// Init may panic on an invalid reporting interval, as described in its documentation.
func MustInitFromViper() {
	var config Config
	viper.MustUnmarshalKey("metrics", &config)

	Init(config)
}

// Init inits metrics with provided metrics configurations.
// Call it only once during application startup, before creating or using metrics.
// An enabled InfluxDB reporter runs for the lifetime of the process and cannot be
// stopped through this API. Reinitialization and dynamic disabling are unsupported:
// another enabled call starts another reporter, and disabling does not stop an existing one.
// It panics if InfluxDB reporting is enabled and ReportInterval is not positive.
// Reporter startup is asynchronous; returning does not guarantee that its client
// was initialized successfully. Client creation failures are logged and stop the
// reporter without terminating the process.
func Init(config Config) {
	if config.Enabled && config.InfluxDb != nil && config.ReportInterval <= 0 {
		panic("metrics: ReportInterval must be positive when InfluxDB reporting is enabled")
	}

	metrics.UseNilMetrics = !config.Enabled

	if !config.Enabled {
		return
	}

	// starts a InfluxDB reporter if configured
	if config.InfluxDb != nil {
		go report.InfluxDB(*config.InfluxDb, metrics.DefaultRegistry, config.ReportInterval)
	}

	logrus.WithField("config", fmt.Sprintf("%+v", config)).Debug("Metrics initialized")
}

// GetOrRegisterCounter gets an existed or registers a new counter from
// default registry by a specified metrics name.
func GetOrRegisterCounter(nameFormat string, nameArgs ...interface{}) metrics.Counter {
	name := fmt.Sprintf(nameFormat, nameArgs...)
	return metrics.GetOrRegisterCounter(name, nil)
}

// GetOrRegisterGauge gets an existed or registers a new gauge from
// default registry by a specified metrics name.
func GetOrRegisterGauge(nameFormat string, nameArgs ...interface{}) metrics.Gauge {
	name := fmt.Sprintf(nameFormat, nameArgs...)
	return metrics.GetOrRegisterGauge(name, nil)
}

// GetOrRegisterGaugeFloat64 gets an existed or registers a new gauge
// from default registry by a specified metrics name.
func GetOrRegisterGaugeFloat64(nameFormat string, nameArgs ...interface{}) metrics.GaugeFloat64 {
	name := fmt.Sprintf(nameFormat, nameArgs...)
	return metrics.GetOrRegisterGaugeFloat64(name, nil)
}

// GetOrRegisterMeter gets an existed or registers a new meter from
// default registry by a specified metrics name.
func GetOrRegisterMeter(nameFormat string, nameArgs ...interface{}) metrics.Meter {
	name := fmt.Sprintf(nameFormat, nameArgs...)
	return metrics.GetOrRegisterMeter(name, nil)
}

// GetOrRegisterHistogram gets an existed or registers a new histogram from
// default registry by a specified metrics name.
func GetOrRegisterHistogram(nameFormat string, nameArgs ...interface{}) metrics.Histogram {
	name := fmt.Sprintf(nameFormat, nameArgs...)
	return metrics.GetOrRegisterHistogram(name, nil, metrics.NewExpDecaySample(expDecaySampleReservoirSize, expDecaySampleAlpha))
}

// GetOrRegisterTimer gets an existed or registers a new timer from
// default registry by a specified metrics name.
func GetOrRegisterTimer(nameFormat string, nameArgs ...interface{}) metrics.Timer {
	name := fmt.Sprintf(nameFormat, nameArgs...)
	return metrics.GetOrRegisterTimer(name, nil)
}

func getOrRegister[T any](factory func() T, name string, args ...interface{}) T {
	name = fmt.Sprintf(name, args...)
	return metrics.DefaultRegistry.GetOrRegister(name, factory).(T)
}
