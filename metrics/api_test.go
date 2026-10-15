package metrics

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMetricsNonFiniteValues(t *testing.T) {
	originalUseNilMetrics := metrics.UseNilMetrics
	metrics.UseNilMetrics = false
	t.Cleanup(func() { metrics.UseNilMetrics = originalUseNilMetrics })

	registry := metrics.NewRegistry()
	percentage := NewPercentage()
	require.NoError(t, registry.Register("test.percentage", &percentageGauge{
		Percentage:          percentage,
		gaugeFloat64Wrapper: gaugeFloat64Wrapper[Percentage]{inner: percentage},
	}))
	for name, value := range map[string]float64{
		"test.nan":    math.NaN(),
		"test.posinf": math.Inf(1),
		"test.neginf": math.Inf(-1),
		"test.finite": 12.5,
	} {
		gauge := metrics.NewGaugeFloat64()
		gauge.Update(value)
		require.NoError(t, registry.Register(name, gauge))
	}
	counter := metrics.NewCounter()
	counter.Inc(7)
	require.NoError(t, registry.Register("other.counter", counter))
	api := NewAPI(registry)

	result, err := api.GetMetrics()
	require.NoError(t, err)
	encoded, err := json.Marshal(result)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"test.percentage":{"value":null},
		"test.nan":{"value":null},
		"test.posinf":{"value":null},
		"test.neginf":{"value":null},
		"test.finite":{"value":12.5},
		"other.counter":{"count":7}
	}`, string(encoded))
	assert.True(t, math.IsNaN(percentage.Value()), "output conversion must not change the metric")
	assert.True(t, math.IsInf(registry.Get("test.posinf").(metrics.GaugeFloat64).Value(), 1))

	percentage.Mark(true)
	result, err = api.GetMetrics("test.")
	require.NoError(t, err)
	assert.Len(t, result, 5)
	assert.NotContains(t, result, "other.counter")
	assert.Equal(t, 100.0, result["test.percentage"]["value"])
	_, err = json.Marshal(result)
	assert.NoError(t, err)
}
