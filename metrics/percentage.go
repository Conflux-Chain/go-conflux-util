package metrics

import (
	"math"
	"sync"

	"github.com/rcrowley/go-metrics"
)

// Percentage implements the GaugeFloat64 interface for percentage statistic.
// The value will be in range [0, 100], e.g. 99.38 means 99.38%.
type Percentage interface {
	Mark(marked bool)

	HasValue[float64]
}

// NewPercentage constructs a new standard percentage metric.
func NewPercentage() Percentage {
	if metrics.UseNilMetrics {
		return &noopPercentage{}
	}

	return &standardPercentage{}
}

var _ metrics.GaugeFloat64 = (*percentageGauge)(nil)

type percentageGauge struct {
	Percentage
	gaugeFloat64Wrapper[Percentage]
}

// GetOrRegisterPercentage returns an existing Percentage or constructs and registers a new standard Percentage.
func GetOrRegisterPercentage(name string, args ...interface{}) Percentage {
	return getOrRegister(func() Percentage {
		percentage := NewPercentage()

		return &percentageGauge{
			Percentage: percentage,
			gaugeFloat64Wrapper: gaugeFloat64Wrapper[Percentage]{
				inner: percentage,
			},
		}
	}, name, args...)
}

// noopPercentage is no-op implementation for Percentage interface.
type noopPercentage struct{}

func (p *noopPercentage) Mark(marked bool) { /* noop */ }
func (p *noopPercentage) Value() float64   { return 0 }

type percentageData struct {
	total uint64
	marks uint64
}

func (data *percentageData) update(marked bool) {
	data.total++
	if marked {
		data.marks++
	}
}

func (data *percentageData) value() float64 {
	if data.total == 0 {
		return math.NaN()
	}

	// usually, data.marks * 10000 will not overflow uint64
	return float64(data.marks*10000/data.total) / 100
}

// standardPercentage is the standard implementation for Percentage interface.
type standardPercentage struct {
	data percentageData

	mu sync.Mutex
}

// Mark implements the Percentage interface.
func (p *standardPercentage) Mark(marked bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.data.update(marked)
}

// Value implements the Percentage interface.
func (p *standardPercentage) Value() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.data.value()
}
