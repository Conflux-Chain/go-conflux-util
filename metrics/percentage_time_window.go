package metrics

import (
	"time"

	"github.com/rcrowley/go-metrics"
)

// NewTimeWindowPercentage constructs a new time window Percentage.
func NewTimeWindowPercentage(slotInterval time.Duration, numSlots int) Percentage {
	if slotInterval == 0 {
		panic("slotInterval is zero")
	}

	if numSlots <= 1 {
		panic("numSlots too small")
	}

	if metrics.UseNilMetrics {
		return &noopPercentage{}
	}

	return &timeWindowPercentage{
		window: NewTimeWindow(slotInterval, numSlots, percentageDataAggregator{}),
	}
}

// GetOrRegisterTimeWindowPercentageDefault returns an existing Percentage or constructs and
// registers a new time window Percentage.
func GetOrRegisterTimeWindowPercentageDefault(name string, args ...interface{}) Percentage {
	return GetOrRegisterTimeWindowPercentage(time.Minute, 10, name, args...)
}

// GetOrRegisterTimeWindowPercentage returns an existing Percentage or constructs and registers
// a new time window Percentage.
func GetOrRegisterTimeWindowPercentage(slotInterval time.Duration, numSlots int, name string, args ...interface{}) Percentage {
	return getOrRegister(func() Percentage {
		percentage := NewTimeWindowPercentage(slotInterval, numSlots)

		return &percentageGauge{
			Percentage: percentage,
			gaugeFloat64Wrapper: gaugeFloat64Wrapper[Percentage]{
				inner: percentage,
			},
		}
	}, name, args...)
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
	window *TimeWindow[percentageData]
}

// Mark implements the Percentage interface.
func (p *timeWindowPercentage) Mark(marked bool) {
	var data percentageData
	data.update(marked)
	p.window.Add(data)
}

// Value implements the Percentage interface.
func (p *timeWindowPercentage) Value() float64 {
	data := p.window.Data()
	return data.value()
}
