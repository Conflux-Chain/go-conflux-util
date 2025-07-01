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
	// interval to report metrics to influxdb
	ReportInterval time.Duration `default:"10s"`
	// settings for influxdb to be reported to
	InfluxDb *report.InfluxDBConfig
}

// MustInitFromViper inits metrics from viper settings.
// This should be called before any metric (e.g. timer, histogram) created.
// Because `metrics.Enabled` in go-ethereum is `false` by default, which leads to noop
// metric created for static variables in any package.
//
// Note that viper must be initialized before this, otherwise metrics
// settings may not be loaded correctly. Besides, this init will panic
// and exit if any error happens.
func MustInitFromViper() {
	var config Config
	viper.MustUnmarshalKey("metrics", &config)

	Init(config)
}

// Init inits metrics with provided metrics configurations.
func Init(config Config) {
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
