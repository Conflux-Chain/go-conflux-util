package metrics

import (
	"math"

	"github.com/rcrowley/go-metrics"
)

type HasValue[T any] interface {
	Value() T
}

type gaugeFloat64Wrapper[T HasValue[float64]] struct {
	inner T
}

// Snapshot implements the metrics.GaugeFloat64 interface.
func (w *gaugeFloat64Wrapper[T]) Snapshot() metrics.GaugeFloat64 {
	return metrics.GaugeFloat64Snapshot(w.inner.Value())
}

// Update implements the metrics.GaugeFloat64 interface.
func (w *gaugeFloat64Wrapper[T]) Update(float64) {
	panic("Unsupported")
}

// Report implements the report.Reportable interface.
func (w *gaugeFloat64Wrapper[T]) Report() (metricType string, fields map[string]interface{}) {
	value := w.inner.Value()
	if math.IsNaN(value) {
		return "gauge", nil
	}

	return "gauge", map[string]interface{}{
		"value": value,
	}
}
