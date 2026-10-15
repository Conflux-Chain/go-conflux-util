package metrics

import (
	"math"
	"testing"
	"time"

	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
)

func TestTimeWindowPercentageParameters(t *testing.T) {
	originalUseNilMetrics := metrics.UseNilMetrics
	t.Cleanup(func() { metrics.UseNilMetrics = originalUseNilMetrics })

	for _, disabled := range []bool{false, true} {
		name := "enabled"
		if disabled {
			name = "disabled"
		}
		t.Run(name, func(t *testing.T) {
			metrics.UseNilMetrics = disabled
			for _, tc := range []struct {
				name     string
				interval time.Duration
				slots    int
				invalid  bool
			}{
				{"zero interval", 0, 2, true},
				{"negative interval", -time.Second, 2, true},
				{"zero slots", time.Second, 0, true},
				{"negative slots", time.Second, -1, true},
				{"single slot", time.Second, 1, true},
				{"duration overflow", time.Duration(math.MaxInt64/2 + 1), 2, true},
				{"two slots", time.Second, 2, false},
				{"maximum two-slot interval", time.Duration(math.MaxInt64 / 2), 2, false},
			} {
				t.Run(tc.name, func(t *testing.T) {
					construct := func() { NewTimeWindowPercentage(tc.interval, tc.slots) }
					if tc.invalid {
						assert.Panics(t, construct)
					} else {
						assert.NotPanics(t, construct)
					}
				})
			}
		})
	}
}
