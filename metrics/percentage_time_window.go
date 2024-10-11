package metrics

import (
	"time"

	"github.com/ethereum/go-ethereum/metrics"
)

// NewTimeWindowPercentage constructs a new time window Percentage with optional default value in range [0, 100].
func NewTimeWindowPercentage(slotInterval time.Duration, numSlots int, defaultValue ...float64) Percentage {
	if slotInterval == 0 {
		panic("slotInterval is zero")
	}

	if numSlots <= 1 {
		panic("numSlots too small")
	}

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

	return &timeWindowPercentage{
		window:       NewTimeWindow(slotInterval, numSlots, percentageDataAggregator{}),
		defaultValue: val,
	}
}

// GetOrRegisterTimeWindowPercentageDefault returns an existing Percentage or constructs and
// registers a new time window Percentage.
func GetOrRegisterTimeWindowPercentageDefault(defaultValue float64, name string, args ...interface{}) Percentage {
	factory := func() Percentage {
		return NewTimeWindowPercentage(time.Minute, 10, defaultValue)
	}

	return getOrRegisterPercentage(factory, name, args...)
}

// GetOrRegisterTimeWindowPercentage returns an existing Percentage or constructs and registers
// a new time window Percentage.
func GetOrRegisterTimeWindowPercentage(
	defaultValue float64, slotInterval time.Duration, numSlots int, name string, args ...interface{},
) Percentage {
	factory := func() Percentage {
		return NewTimeWindowPercentage(slotInterval, numSlots, defaultValue)
	}

	return getOrRegisterPercentage(factory, name, args...)
}

type percentageDataAggregator struct{}

// Add implements the SlotAggregator[percentageData] interface.
func (percentageDataAggregator) Add(acc, v percentageData) percentageData {
	return percentageData{
		total: acc.total + v.total,
		marks: acc.marks + v.marks,
	}
}

// Sub implements the SlotAggregator[percentageData] interface.
func (percentageDataAggregator) Sub(acc, v percentageData) percentageData {
	return percentageData{
		total: acc.total - v.total,
		marks: acc.marks - v.marks,
	}
}

// timeWindowPercentage implements Percentage interface to record recent percentage.
type timeWindowPercentage struct {
	window       *TimeWindow[percentageData]
	defaultValue float64
}

// Mark implements the Percentage interface.
func (p *timeWindowPercentage) Mark(marked bool) {
	var data percentageData
	data.update(marked)
	p.window.Add(data)
}

// Update implements the metrics.GaugeFloat64 interface.
func (p *timeWindowPercentage) Update(float64) {
	panic("Update called on a timeWindowPercentage")
}

// Snapshot implements the metrics.GaugeFloat64 interface.
func (p *timeWindowPercentage) Snapshot() metrics.GaugeFloat64Snapshot {
	data := p.window.Data()
	return data.toSnapshot(p.defaultValue)
}
