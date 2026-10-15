package metrics

import (
	"maps"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mapSlotAggregator struct{}

func (mapSlotAggregator) Add(acc, v map[string]int) map[string]int {
	if acc == nil {
		acc = make(map[string]int)
	}
	for key, value := range v {
		acc[key] += value
	}
	return acc
}

func (mapSlotAggregator) Sub(acc, v map[string]int) map[string]int {
	for key, value := range v {
		acc[key] -= value
	}
	return acc
}

func (mapSlotAggregator) Clone(v map[string]int) map[string]int {
	return maps.Clone(v)
}

func TestTimeWindowDoesNotMutateMapSample(t *testing.T) {
	for _, tc := range []struct {
		name string
		new  func() *TimeWindow[map[string]int]
	}{
		{"standard", func() *TimeWindow[map[string]int] {
			return NewTimeWindow(time.Second, 2, mapSlotAggregator{})
		}},
		{"cloneable", func() *TimeWindow[map[string]int] {
			return NewTimeWindowCloneable(time.Second, 2, mapSlotAggregator{})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			window := tc.new()
			start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			sample := map[string]int{"requests": 1}
			window.add(start.Add(100*time.Millisecond), sample)
			window.add(start.Add(200*time.Millisecond), sample)
			assert.Equal(t, 1, sample["requests"])
			assert.Equal(t, 2, window.data(start.Add(300 * time.Millisecond))["requests"])

			// Reusing the sample in another slot must also keep the slots independent.
			window.add(start.Add(1100*time.Millisecond), sample)
			assert.Equal(t, 1, sample["requests"])
			assert.Equal(t, 3, window.data(start.Add(1200 * time.Millisecond))["requests"])
			assert.Equal(t, 1, window.data(start.Add(2 * time.Second))["requests"])
			assert.Equal(t, 0, window.data(start.Add(3 * time.Second))["requests"])
		})
	}
}

func TestTimeWindowParameters(t *testing.T) {
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
		{"overflow to negative", time.Duration(math.MaxInt64/2 + 1), 2, true},
		{"overflow to zero", time.Duration(1 << 62), 4, true},
		{"overflow to positive", time.Duration(math.MaxInt64), 3, true},
		{"single slot", time.Second, 1, false},
		{"minimum interval", time.Nanosecond, 2, false},
		{"maximum duration", time.Duration(math.MaxInt64), 1, false},
		{"maximum two-slot interval", time.Duration(math.MaxInt64 / 2), 2, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			construct := func() {
				window := NewSimpleTimeWindow[int](tc.interval, tc.slots)
				assert.Equal(t, tc.interval*time.Duration(tc.slots), window.windowInterval)
			}
			if tc.invalid {
				assert.Panics(t, construct)
			} else {
				assert.NotPanics(t, construct)
			}
		})
	}
}

func TestTimeWindowAddNewSlot(t *testing.T) {
	timeWin := NewSimpleTimeWindow[int](time.Second, 5)
	startT := time.Now()

	slot := timeWin.addNewSlot(startT, 3)

	assert.Equal(t, 1, timeWin.slots.Len())
	assert.Equal(t, 3, slot.data)

	assert.True(t, !slot.expired(startT))
	assert.True(t, !slot.outdated(startT))

	testT := startT.Add(time.Second)
	assert.True(t, slot.outdated(testT))
	assert.True(t, !slot.expired(testT))

	testT = startT.Add(time.Second * 5)
	assert.True(t, slot.outdated(testT))
	assert.True(t, slot.expired(testT))
}

func TestTimeWindowExpire(t *testing.T) {
	timeWin := NewSimpleTimeWindow[int](time.Second, 5)

	startT := time.Now()
	slot := timeWin.addNewSlot(startT, 3)

	testT := startT.Add(time.Second * 5)
	expSlots := timeWin.expire(testT)

	assert.Equal(t, 1, len(expSlots))
	assert.Equal(t, expSlots[0], slot)

	assert.Equal(t, timeWin.slots.Len(), 0)
}

func TestTimeWindowAddOrUpdateSlot(t *testing.T) {
	timeWin := NewSimpleTimeWindow[int](time.Second, 5)
	startT := time.Now()

	slot1, added := timeWin.addOrUpdateSlot(startT, 3)

	assert.True(t, added)
	assert.Equal(t, slot1.data, 3)

	slot1, added = timeWin.addOrUpdateSlot(startT, 4)
	assert.False(t, added)
	assert.Equal(t, slot1.data, 7)

	testT := startT.Add(time.Second)
	slot2, added := timeWin.addOrUpdateSlot(testT, 3)
	assert.True(t, added)
	assert.NotEqual(t, slot1, slot2)
}

func TestTimeWindowAdd(t *testing.T) {
	timeWin := NewSimpleTimeWindow[int](time.Second, 5)
	startT := time.Now()

	timeWin.add(startT, 3)
	assert.Equal(t, 3, timeWin.data(startT))

	testT := startT.Add(time.Second * 2)

	timeWin.add(testT, 4)
	assert.Equal(t, 7, timeWin.data(testT))

	testT = startT.Add(time.Second * 6)
	assert.Equal(t, 4, timeWin.data(testT))
}

func TestSlotTimeBoundaries(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s := slot[int]{endTime: start.Add(time.Second), expireAt: start.Add(2 * time.Second)}

	for _, tc := range []struct {
		name   string
		offset time.Duration
		want   bool
	}{
		{"before", -time.Nanosecond, false},
		{"at", 0, true},
		{"after", time.Nanosecond, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, s.outdated(s.endTime.Add(tc.offset)))
			assert.Equal(t, tc.want, s.expired(s.expireAt.Add(tc.offset)))
		})
	}
}

func TestTimeWindowBoundarySample(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	window := NewSimpleTimeWindow[int](time.Second, 2)
	window.add(start.Add(500*time.Millisecond), 1)
	window.add(start.Add(time.Second), 10)

	assert.Equal(t, 2, window.slots.Len())
	assert.Equal(t, 11, window.data(start.Add(2*time.Second-time.Nanosecond)))
	assert.Equal(t, 10, window.data(start.Add(2*time.Second)))
	assert.Equal(t, 10, window.data(start.Add(2250*time.Millisecond)))
	assert.Equal(t, 0, window.data(start.Add(3*time.Second)))
}
