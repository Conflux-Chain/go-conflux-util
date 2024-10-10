package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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
