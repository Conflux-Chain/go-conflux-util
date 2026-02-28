package poll

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReorgWindowPush(t *testing.T) {
	window := NewReorgWindow()

	// push empty
	pushed, popped := window.Push(5, "Hash - 5", "Hash - 4")
	assert.Equal(t, true, pushed)
	assert.Equal(t, false, popped)

	// push in sequence
	pushed, popped = window.Push(6, "Hash - 6", "Hash - 5")
	assert.Equal(t, true, pushed)
	assert.Equal(t, false, popped)

	// push not in sequence
	pushed, popped = window.Push(6, "Hash - 6", "Hash - 5")
	assert.Equal(t, false, pushed)
	assert.Equal(t, false, popped)
	pushed, popped = window.Push(8, "Hash - 8", "Hash - 7")
	assert.Equal(t, false, pushed)
	assert.Equal(t, false, popped)

	// push in sequence, but parent hash mismatch
	pushed, popped = window.Push(7, "Hash - 7", "Hash - 66")
	assert.Equal(t, false, pushed)
	assert.Equal(t, true, popped)

	// push block 6 again due to popped
	pushed, popped = window.Push(6, "Hash - 66", "Hash - 5")
	assert.Equal(t, true, pushed)
	assert.Equal(t, false, popped)
}

func TestReorgWindowEvict(t *testing.T) {
	window := NewReorgWindow()

	for i := uint64(1); i < 10; i++ {
		pushed, popped := window.Push(i, fmt.Sprintf("Hash - %v", i), fmt.Sprintf("Hash - %v", i-1))
		assert.Equal(t, true, pushed)
		assert.Equal(t, false, popped)
	}

	window.Evict(5)

	assert.Equal(t, uint64(6), window.earliest)
}

func TestReorgWindowWithLatestBlocks(t *testing.T) {
	window := NewReorgWindowWithLatestBlocks(ReorgWindowParams{
		FinalizedBlockNumber: 5,
		FinalizedBlockHash:   "Hash - 5",
		LatestBlocks: map[uint64]string{
			6: "Hash - 6",
			7: "Hash - 7",
			8: "Hash - 8",
		},
	})

	assert.Equal(t, uint64(5), window.earliest)
	assert.Equal(t, uint64(8), window.latest)

	pushed, popped := window.Push(9, "Hash - 9", "Hash - 8")
	assert.Equal(t, true, pushed)
	assert.Equal(t, false, popped)
}
