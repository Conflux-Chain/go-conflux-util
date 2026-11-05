package poll

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReorgWindowPush(t *testing.T) {
	window, err := NewReorgWindow(ReorgWindowParams{})
	require.NoError(t, err)
	assert.Equal(t, uint64(0), window.NextBlockNumber())

	// An empty window rejects a non-genesis start without changing state.
	pushed, popped, err := window.Push(5, "Hash - 5", "Hash - 4")
	assert.Error(t, err)
	assert.False(t, pushed)
	assert.False(t, popped)
	assert.Empty(t, window.blockNumber2Hashes)

	// Genesis establishes the anchor without checking its parent hash.
	pushed, popped, err = window.Push(0, "Hash - 0", "ignored")
	assert.NoError(t, err)
	assert.Equal(t, true, pushed)
	assert.Equal(t, false, popped)
	assert.Equal(t, uint64(1), window.NextBlockNumber())

	// push in sequence
	pushed, popped, err = window.Push(1, "Hash - 1", "Hash - 0")
	assert.NoError(t, err)
	assert.Equal(t, true, pushed)
	assert.Equal(t, false, popped)
	assert.Equal(t, uint64(2), window.NextBlockNumber())

	// push not in sequence
	_, _, err = window.Push(1, "Hash - 1", "Hash - 0")
	assert.Error(t, err)
	_, _, err = window.Push(3, "Hash - 3", "Hash - 2")
	assert.Error(t, err)

	// push in sequence, but parent hash mismatch
	pushed, popped, err = window.Push(2, "Hash - 2", "Hash - 11")
	assert.NoError(t, err)
	assert.Equal(t, false, pushed)
	assert.Equal(t, true, popped)
	assert.Equal(t, uint64(1), window.NextBlockNumber())

	// The genesis anchor cannot be reverted.
	_, _, err = window.Push(1, "Hash - 11", "Hash - 00")
	assert.Error(t, err)

	// push block 1 again
	pushed, popped, err = window.Push(1, "Hash - 11", "Hash - 0")
	assert.NoError(t, err)
	assert.Equal(t, true, pushed)
	assert.Equal(t, false, popped)
}

func TestReorgWindowEvict(t *testing.T) {
	window, err := NewReorgWindow(ReorgWindowParams{})
	require.NoError(t, err)

	// evict empty window
	assert.Equal(t, 0, window.Evict(5, "Hash - 5"))

	// push block 0 - 9
	for i := uint64(0); i <= 9; i++ {
		pushed, popped, err := window.Push(i, fmt.Sprintf("Hash - %v", i), fmt.Sprintf("Hash - %v", i-1))
		assert.NoError(t, err)
		assert.Equal(t, true, pushed)
		assert.Equal(t, false, popped)
	}

	// finalized block == 5, evict block 0 - 4.
	assert.Equal(t, 5, window.Evict(5, "Hash - 5"))
	assert.Len(t, window.blockNumber2Hashes, 5)
	assert.Equal(t, uint64(5), window.finalized)
	assert.Equal(t, "Hash - 5", window.blockNumber2Hashes[5])

	// Stale heights, heights outside the window and another fork cannot advance finality.
	assert.Equal(t, 0, window.Evict(4, "Hash - 4"))
	assert.Equal(t, 0, window.Evict(5, "Hash - 5"))
	assert.Equal(t, 0, window.Evict(100, "Hash - 100"))
	assert.Equal(t, 0, window.Evict(9, "Hash - 99"))
	assert.Len(t, window.blockNumber2Hashes, 5)
	assert.Equal(t, uint64(5), window.finalized)

	// A matching finalized tip removes blocks 5-8, keeping block 9 as the anchor.
	assert.Equal(t, 4, window.Evict(9, "Hash - 9"))
	assert.Equal(t, uint64(9), window.finalized)
	assert.Equal(t, map[uint64]string{9: "Hash - 9"}, window.blockNumber2Hashes)
}

func TestReorgWindowWithLatestBlocks(t *testing.T) {
	for _, tc := range []struct {
		name   string
		params ReorgWindowParams
	}{
		{"height without anchor hash", ReorgWindowParams{FinalizedBlockNumber: 5}},
		{"blocks without anchor hash", ReorgWindowParams{LatestBlocks: map[uint64]string{1: "Hash - 1"}}},
		{"gap after anchor", ReorgWindowParams{FinalizedBlockNumber: 5, FinalizedBlockHash: "Hash - 5", LatestBlocks: map[uint64]string{7: "Hash - 7"}}},
		{"empty block hash", ReorgWindowParams{FinalizedBlockNumber: 5, FinalizedBlockHash: "Hash - 5", LatestBlocks: map[uint64]string{6: ""}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			window, err := NewReorgWindow(tc.params)
			assert.Error(t, err)
			assert.Nil(t, window)
		})
	}

	window, err := NewReorgWindow(ReorgWindowParams{
		FinalizedBlockNumber: 5,
		FinalizedBlockHash:   "Hash - 5",
		LatestBlocks: map[uint64]string{
			6: "Hash - 6",
			7: "Hash - 7",
			8: "Hash - 8",
		},
	})
	assert.NoError(t, err)

	// A matching finalized hash verifies the restored prefix without a new Push.
	assert.Equal(t, 0, window.Evict(100, "Hash - 100"))
	assert.Equal(t, 0, window.Evict(7, "Hash - 77"))
	assert.Equal(t, 2, window.Evict(7, "Hash - 7"))
	assert.Equal(t, uint64(7), window.finalized)
	assert.Equal(t, map[uint64]string{7: "Hash - 7", 8: "Hash - 8"}, window.blockNumber2Hashes)

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
