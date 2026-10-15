package metrics

import (
	"testing"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/metrics/report"
	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
)

func TestInitReportInterval(t *testing.T) {
	originalUseNilMetrics := metrics.UseNilMetrics
	t.Cleanup(func() { metrics.UseNilMetrics = originalUseNilMetrics })

	for _, interval := range []time.Duration{0, -time.Second} {
		t.Run(interval.String(), func(t *testing.T) {
			metrics.UseNilMetrics = true
			config := Config{
				Enabled:        true,
				ReportInterval: interval,
				InfluxDb:       &report.InfluxDBConfig{Host: "http://127.0.0.1:8086"},
			}
			assert.PanicsWithValue(t,
				"metrics: ReportInterval must be positive when InfluxDB reporting is enabled",
				func() { Init(config) })
			assert.True(t, metrics.UseNilMetrics, "invalid configuration must not change global state")

			config.Enabled = false
			assert.NotPanics(t, func() { Init(config) })
			assert.True(t, metrics.UseNilMetrics)

			config.Enabled = true
			config.InfluxDb = nil
			assert.NotPanics(t, func() { Init(config) })
			assert.False(t, metrics.UseNilMetrics)
		})
	}
}
