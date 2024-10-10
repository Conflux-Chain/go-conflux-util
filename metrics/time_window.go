package metrics

import (
	"container/list"
	"sync"
	"time"
)

// SlotAggregator defines operators for slot data.
type SlotAggregator[T any] interface {
	// Add returns the sum x+y. Note, x may be nil if T is pointer type.
	Add(x, y T) T

	// Sub returns the difference x-y.
	Sub(x, y T) T
}

type SimpleSlotData interface {
	int | int64 | uint | uint64 | float32 | float64
}

type simpleSlotAggregator[T SimpleSlotData] struct{}

// Add implements the SlotAggregator[T] interface.
func (simpleSlotAggregator[T]) Add(x, y T) T {
	return x + y
}

// Sub implements the SlotAggregator[T] interface.
func (simpleSlotAggregator[T]) Sub(x, y T) T {
	return x - y
}

// time window slot
type slot[T any] struct {
	data     T         // slot data
	endTime  time.Time // end time for slot update
	expireAt time.Time // expiry time to remove
}

// check if slot expired (can be purged)
func (s slot[T]) expired(now time.Time) bool {
	return s.expireAt.Before(now)
}

// check if slot outdated (not open for update)
func (s slot[T]) outdated(now time.Time) bool {
	return s.endTime.Before(now)
}

// TimeWindow slices time window into slots and maintains slot expiry and creation
type TimeWindow[T any] struct {
	mu sync.Mutex

	slots          *list.List    // double linked slots chronologically
	slotInterval   time.Duration // time interval per slot
	windowInterval time.Duration // time window interval

	aggData    T                 // aggregation data within the time window scope
	aggregator SlotAggregator[T] // to aggregate slot data
}

func NewTimeWindow[T any](slotInterval time.Duration, numSlots int, aggregator SlotAggregator[T], val ...T) *TimeWindow[T] {
	tw := TimeWindow[T]{
		slots:          list.New(),
		slotInterval:   slotInterval,
		windowInterval: slotInterval * time.Duration(numSlots),
		aggregator:     aggregator,
	}

	if len(val) > 0 {
		tw.aggData = val[0]
	}

	return &tw
}

func NewSimpleTimeWindow[T SimpleSlotData](slotInterval time.Duration, numSlots int, val ...T) *TimeWindow[T] {
	return NewTimeWindow(slotInterval, numSlots, simpleSlotAggregator[T]{}, val...)
}

// Add adds data sample to time window
func (tw *TimeWindow[T]) Add(sample T) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	tw.add(time.Now(), sample)
}

func (tw *TimeWindow[T]) add(now time.Time, sample T) {
	// expire outdated slots
	tw.expire(now)

	// add or update slot data
	tw.addOrUpdateSlot(now, sample)

	// update agg data
	tw.aggData = tw.aggregator.Add(tw.aggData, sample)
}

// Data returns the aggregation data within the time window scope
func (tw *TimeWindow[T]) Data() T {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	return tw.data(time.Now())
}

func (tw *TimeWindow[T]) data(now time.Time) T {
	// expire outdated slots
	tw.expire(now)

	return tw.aggData
}

// expire removes expired slots.
func (tw *TimeWindow[T]) expire(now time.Time) (res []*slot[T]) {
	for {
		// time window is empty
		front := tw.slots.Front()
		if front == nil {
			return res
		}

		// not expired yet
		s := front.Value.(*slot[T])
		if !s.expired(now) {
			return res
		}

		// remove expired slot
		tw.slots.Remove(front)
		res = append(res, s)

		// dissipate expired slot data
		tw.aggData = tw.aggregator.Sub(tw.aggData, s.data)
	}
}

// addOrUpdateSlot adds a new slot with the provided slot data if no one exists or
// the last one is out of date; otherwise update the last slot with the provided data.
func (tw *TimeWindow[T]) addOrUpdateSlot(now time.Time, data T) (*slot[T], bool) {
	// time window is empty
	if tw.slots.Len() == 0 {
		return tw.addNewSlot(now, data), true
	}

	// last slot is out of date
	lastSlot := tw.slots.Back().Value.(*slot[T])
	if lastSlot.outdated(now) {
		return tw.addNewSlot(now, data), true
	}

	// otherwise, update the last slot with new data
	lastSlot.data = tw.aggregator.Add(lastSlot.data, data)
	return lastSlot, false
}

// addNewSlot always appends a new slot to time window.
func (tw *TimeWindow[T]) addNewSlot(now time.Time, data T) *slot[T] {
	slotStartTime := now.Truncate(tw.slotInterval)

	newSlot := &slot[T]{
		data:     data,
		endTime:  slotStartTime.Add(tw.slotInterval),
		expireAt: slotStartTime.Add(tw.windowInterval),
	}

	tw.slots.PushBack(newSlot)
	return newSlot
}
