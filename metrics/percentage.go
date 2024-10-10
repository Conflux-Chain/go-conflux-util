package metrics

import (
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/metrics"
)

// Percentage implements the GaugeFloat64 interface for percentage statistic.
// The value will be in range [0, 100], e.g. 99.38 means 99.38%.
type Percentage interface {
	metrics.GaugeFloat64

	Mark(marked bool)
}

// NewPercentage constructs a new standard percentage metric with optional default value in range [0, 100].
func NewPercentage(defaultValue ...float64) Percentage {
	if !metrics.Enabled {
		return &noopPercentage{}
	}

	var val float64
	if len(defaultValue) > 0 {
		val = defaultValue[0]
	}

	if val < 0 || val > 100 {
		panic("default value should be in range [0, 100]")
	}

	return &standardPercentage{
		defaultValue: val,
	}
}

// GetOrRegisterPercentage returns an existing Percentage or constructs and registers a new standard Percentage.
func GetOrRegisterPercentage(defaultValue float64, name string, args ...interface{}) Percentage {
	factory := func() Percentage {
		return NewPercentage(defaultValue)
	}

	return getOrRegisterPercentage(factory, name, args...)
}

// getOrRegisterPercentage gets or constructs Percentage with specified factory.
func getOrRegisterPercentage(factory func() Percentage, name string, args ...interface{}) Percentage {
	metricName := fmt.Sprintf(name, args...)
	return DefaultRegistry.GetOrRegister(metricName, factory).(Percentage)
}

// noopPercentage is no-op implementation for Percentage interface.
type noopPercentage struct{}

func (p *noopPercentage) Mark(marked bool)                       { /* noop */ }
func (p *noopPercentage) Update(float64)                         { /* noop */ }
func (p *noopPercentage) Snapshot() metrics.GaugeFloat64Snapshot { return percentageSnapshot(0) }

type percentageSnapshot float64

// Value implements the metrics.GaugeFloat64Snapshot interface.
func (s percentageSnapshot) Value() float64 {
	return float64(s)
}

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

func (data *percentageData) toSnapshot(defaultValue float64) metrics.GaugeFloat64Snapshot {
	if data.total == 0 {
		return percentageSnapshot(defaultValue)
	}

	// 10.19 means 10.19%
	val := float64(data.marks*10000/data.total) / 100

	return percentageSnapshot(val)
}

// standardPercentage is the standard implementation for Percentage interface.
type standardPercentage struct {
	data         percentageData
	defaultValue float64
	mu           sync.Mutex
}

// Mark implements the Percentage interface.
func (p *standardPercentage) Mark(marked bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.data.update(marked)
}

// Update implements the metrics.GaugeFloat64 interface.
func (p *standardPercentage) Update(float64) {
	panic("Update called on a standardPercentage")
}

// Snapshot implements the metrics.GaugeFloat64 interface.
func (p *standardPercentage) Snapshot() metrics.GaugeFloat64Snapshot {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.data.toSnapshot(p.defaultValue)
}
