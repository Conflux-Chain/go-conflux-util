package metrics

import (
	"fmt"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/influxdb"
	"github.com/sirupsen/logrus"
)

const (
	// default exponentially-decaying metrics sample reservoir size
	expDecaySampleReservoirSize int = 1028

	// default exponentially-decaying metrics sample alpha
	expDecaySampleAlpha float64 = 0.015
)

var (
	// default metrics registry
	DefaultRegistry = metrics.NewRegistry()
)

// MetricsConfig metrics configurations such as influxdb settings.,
type MetricsConfig struct {
	// switch to turn on or off metrics
	Enabled bool
	// namespace for metrics reporting
	Namespace string
	// interval to report metrics to influxdb
	ReportInterval time.Duration `default:"10s"`
	// settings for influxdb to be reported to
	InfluxDb *InfluxDbConfig
}

// InfluxDbConfig influxdb configurations.
type InfluxDbConfig struct {
	// host path
	Host string `default:"http://127.0.0.1:8086"`
	// database name
	Db string `default:"metrics_db"`
	// authenticated username
	Username string
	// authenticated password
	Password string
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
	var config MetricsConfig
	viper.MustUnmarshalKey("metrics", &config)

	Init(config)
}

// Init inits metrics with provided metrics configurations.
func Init(config MetricsConfig) {
	if !config.Enabled { // metrics not enabled?
		return
	}

	// go-ethereum `metrics.Enabled` must be set, otherwise it will lead to
	// noop metric created for static variables in any package.
	metrics.Enabled = true

	if config.InfluxDb != nil {
		// starts a InfluxDB reporter
		go influxdb.InfluxDB(
			DefaultRegistry,
			config.ReportInterval,
			config.InfluxDb.Host,
			config.InfluxDb.Db,
			config.InfluxDb.Username,
			config.InfluxDb.Password,
			config.Namespace,
		)
	}

	logrus.WithField("config", fmt.Sprintf("%+v", config)).Debug("Metrics initialized")
}

// GetOrRegisterCounter gets an existed or registers a new counter from
// default registry by a specified metrics name.
func GetOrRegisterCounter(name string) metrics.Counter {
	return metrics.GetOrRegisterCounter(name, DefaultRegistry)
}

// GetOrRegisterGauge gets an existed or registers a new gauge from
// default registry by a specified metrics name.
func GetOrRegisterGauge(name string) metrics.Gauge {
	return metrics.GetOrRegisterGauge(name, DefaultRegistry)
}

// GetOrRegisterGaugeFloat64 gets an existed or registers a new gauge
// from default registry by a specified metrics name.
func GetOrRegisterGaugeFloat64(name string) metrics.GaugeFloat64 {
	return metrics.GetOrRegisterGaugeFloat64(name, DefaultRegistry)
}

// GetOrRegisterMeter gets an existed or registers a new meter from
// default registry by a specified metrics name.
func GetOrRegisterMeter(name string) metrics.Meter {
	return metrics.GetOrRegisterMeter(name, DefaultRegistry)
}

// GetOrRegisterHistogram gets an existed or registers a new histogram from
// default registry by a specified metrics name.
func GetOrRegisterHistogram(name string) metrics.Histogram {
	return metrics.GetOrRegisterHistogram(
		name, DefaultRegistry, metrics.NewExpDecaySample(
			expDecaySampleReservoirSize, expDecaySampleAlpha,
		),
	)
}

// GetOrRegisterTimer gets an existed or registers a new timer from
// default registry by a specified metrics name.
func GetOrRegisterTimer(name string) metrics.Timer {
	return metrics.GetOrRegisterTimer(name, DefaultRegistry)
}
