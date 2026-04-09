package poll

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReorgWindowPush(t *testing.T) {
	window := NewReorgWindow()

	// push empty
	pushed, popped, err := window.Push(5, "Hash - 5", "Hash - 4")
	assert.NoError(t, err)
	assert.Equal(t, true, pushed)
	assert.Equal(t, false, popped)

	// push in sequence
	pushed, popped, err = window.Push(6, "Hash - 6", "Hash - 5")
	assert.NoError(t, err)
	assert.Equal(t, true, pushed)
	assert.Equal(t, false, popped)

	// push not in sequence
	_, _, err = window.Push(6, "Hash - 6", "Hash - 5")
	assert.Error(t, err)
	_, _, err = window.Push(8, "Hash - 8", "Hash - 7")
	assert.Error(t, err)

	// push in sequence, but parent hash mismatch
	pushed, popped, err = window.Push(7, "Hash - 7", "Hash - 66")
	assert.NoError(t, err)
	assert.Equal(t, false, pushed)
	assert.Equal(t, true, popped)

	// pop the finalized block 5
	_, _, err = window.Push(6, "Hash - 66", "Hash - 55")
	assert.Error(t, err)

	// push block 6 again
	pushed, popped, err = window.Push(6, "Hash - 66", "Hash - 5")
	assert.NoError(t, err)
	assert.Equal(t, true, pushed)
	assert.Equal(t, false, popped)
}

func TestReorgWindowEvict(t *testing.T) {
	window := NewReorgWindow()

	// evict empty window
	assert.Equal(t, 0, window.Evict(5))

	// push block 1 - 9
	for i := uint64(1); i <= 9; i++ {
		pushed, popped, err := window.Push(i, fmt.Sprintf("Hash - %v", i), fmt.Sprintf("Hash - %v", i-1))
		assert.NoError(t, err)
		assert.Equal(t, true, pushed)
		assert.Equal(t, false, popped)
	}

	// finalized block == 5, evict block 1 - 4.
	assert.Equal(t, 4, window.Evict(5))

	// finalized block is very big, then evit all except the latest block 9
	assert.Equal(t, 4, window.Evict(100))
}

func TestReorgWindowWithLatestBlocks(t *testing.T) {
	window, err := NewReorgWindowWithLatestBlocks(ReorgWindowParams{
		FinalizedBlockNumber: 5,
		FinalizedBlockHash:   "Hash - 5",
		LatestBlocks: map[uint64]string{
			6: "Hash - 6",
			7: "Hash - 7",
			8: "Hash - 8",
		},
	})
	assert.NoError(t, err)

	// evict nothing since not verify against on-chain data
	assert.Equal(t, 0, window.Evict(100))
	assert.Equal(t, 0, window.Evict(7))

	// push block 9, but reorg happened during service down time
	pushed, popped, err := window.Push(9, "Hash - 9", "Hash - 88")
	assert.NoError(t, err)
	assert.Equal(t, false, pushed)
	assert.Equal(t, true, popped)

	// push block 8
	pushed, popped, err = window.Push(8, "Hash - 88", "Hash - 7")
	assert.NoError(t, err)
	assert.Equal(t, true, pushed)
	assert.Equal(t, false, popped)
}
