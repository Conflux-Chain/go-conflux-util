package metrics

import (
	"math"
	"testing"

	"github.com/Conflux-Chain/go-conflux-util/metrics/report"
	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
)

func reset() {
	Init(Config{Enabled: true})

	metrics.DefaultRegistry.UnregisterAll()
}

func TestPercentage(t *testing.T) {
	p := NewPercentage()

	// NaN by default
	value := p.Value()
	assert.True(t, math.IsNaN(value))

	// updated to 33.33
	p.Mark(true)
	p.Mark(false)
	p.Mark(false)

	value = p.Value()
	assert.Equal(t, 33.33, value)
}

func TestGetOrRegisterPercentage(t *testing.T) {
	reset()

	p1 := GetOrRegisterPercentage("p1")
	p1.Mark(true)
	p1.Mark(false)

	// get metric in registry
	p2 := metrics.DefaultRegistry.Get("p1").(Percentage)
	assert.Same(t, p1, p2)
	assert.Equal(t, float64(50), p2.Value())

	// register again
	p3 := GetOrRegisterPercentage("p1")
	assert.Same(t, p1, p3)
	assert.Equal(t, float64(50), p3.Value())
}

func TestPercentageReportable(t *testing.T) {
	reset()

	// NaN by default
	p1 := GetOrRegisterPercentage("p1")
	name, fields := report.ReadMeter("foo-", "bar", p1)
	assert.Equal(t, "foo-bar.gauge", name)
	assert.Nil(t, fields)

	// updated
	p1.Mark(true)
	name, fields = report.ReadMeter("foo-", "bar", p1)
	assert.Equal(t, "foo-bar.gauge", name)
	assert.Equal(t, map[string]any{"value": 100.0}, fields)
}
