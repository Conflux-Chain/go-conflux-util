package poll

import (
	"errors"
	"fmt"
)

// ReorgWindowParams describes a finalized anchor and optional consecutive blocks
// from the same chain. The caller must ensure the anchor is finalized.
// Zero parameters select an empty window for synchronization from genesis.
type ReorgWindowParams struct {
	FinalizedBlockNumber uint64
	FinalizedBlockHash   string
	// LatestBlocks starts at FinalizedBlockNumber + 1, with no gaps or empty hashes.
	LatestBlocks map[uint64]string
}

// ReorgWindow is used to detect chain reorg.
//
// Basically, the reorg window maintains the recent blocks and the latest finalized block.
//
// Entries contain only block numbers and hashes. They are retained until a matching
// finalized block is available within the window; no explicit capacity is required.
type ReorgWindow struct {
	blockNumber2Hashes map[uint64]string // recent blocks in sequence
	finalized          uint64            // the finalized block number, which will not be reverted
	latest             uint64            // the latest block number, which may be reverted
}

// NewReorgWindow restores a window from a finalized anchor and
// optional consecutive blocks after it. The anchor hash and all block hashes
// must be nonempty. Zero parameters create an empty window whose first Push
// must be genesis (block 0), which becomes the initial finalized anchor.
//
// When service restarted, user should load recent blocks and the last finalized block from database,
// and initialize the reorg window so as to correctly handle chain reorg during the service down time.
func NewReorgWindow(params ReorgWindowParams) (*ReorgWindow, error) {
	window := &ReorgWindow{
		blockNumber2Hashes: make(map[uint64]string),
	}

	if len(params.FinalizedBlockHash) == 0 {
		if params.FinalizedBlockNumber != 0 || len(params.LatestBlocks) != 0 {
			return nil, errors.New("Invalid reorg window params to sync from genesis block")
		}
		return window, nil
	}

	// set the finalized block
	window.finalized = params.FinalizedBlockNumber
	window.latest = params.FinalizedBlockNumber
	window.blockNumber2Hashes[params.FinalizedBlockNumber] = params.FinalizedBlockHash

	// continuous blocks required
	for i, numBlocks := 0, len(params.LatestBlocks); i < numBlocks; i++ {
		expectedBlockNumber := params.FinalizedBlockNumber + 1 + uint64(i)

		hash, ok := params.LatestBlocks[expectedBlockNumber]
		if !ok {
			return nil, fmt.Errorf("Block %v missed", expectedBlockNumber)
		}

		if len(hash) == 0 {
			return nil, fmt.Errorf("Block %v hash is empty", expectedBlockNumber)
		}

		window.blockNumber2Hashes[expectedBlockNumber] = hash
		window.latest = expectedBlockNumber
	}

	return window, nil
}

// NextBlockNumber returns the next height to push, or 0 for an empty window.
func (window *ReorgWindow) NextBlockNumber() uint64 {
	if len(window.blockNumber2Hashes) == 0 {
		return 0
	}

	return window.latest + 1
}

// Push pushes the latest block into reorg window.
// An empty window accepts only genesis (block 0), without checking its parent hash.
// Subsequent blocks must follow the window's latest height; a parent hash mismatch
// rolls back one block, but never the finalized anchor.
func (window *ReorgWindow) Push(blockNumber uint64, blockHash, parentBlockHash string) (appended, popped bool, err error) {
	// Initialize the anchor from genesis only.
	if len(window.blockNumber2Hashes) == 0 {
		if blockNumber != 0 {
			return false, false, fmt.Errorf("Empty reorg window must start from genesis, got block %v", blockNumber)
		}

		window.finalized = 0
		window.latest = 0
		window.blockNumber2Hashes[0] = blockHash
		return true, false, nil
	}

	// not in sequence
	if window.latest+1 != blockNumber {
		return false, false, fmt.Errorf("Block not in sequence, latest = %v, new = %v", window.latest, blockNumber)
	}

	// appended if parent block hash matches
	if window.blockNumber2Hashes[window.latest] == parentBlockHash {
		window.blockNumber2Hashes[blockNumber] = blockHash
		window.latest = blockNumber
		return true, false, nil
	}

	// finalized block should never be reverted
	if window.finalized == window.latest {
		return false, false, fmt.Errorf("Finalized block %v should not be reverted", window.finalized)
	}

	// reorg detected, pop the latest block from reorg window
	delete(window.blockNumber2Hashes, window.latest)
	window.latest--

	return false, true, nil
}

// Evict removes blocks before the given finalized block and returns the number removed.
// The given number and hash must identify the same on-chain finalized block.
// It keeps that block as the new finalized anchor and ignores an older or equal
// finalized height, a height beyond the window, a missing window entry, or a hash mismatch.
func (window *ReorgWindow) Evict(number uint64, hash string) (evicted int) {
	if number <= window.finalized || number > window.latest {
		return
	}

	// Restored or previously polled blocks may belong to an old fork. Match the
	// finalized hash before discarding history needed to recover from a reorg.
	cachedHash, ok := window.blockNumber2Hashes[number]
	if !ok || cachedHash != hash {
		return
	}

	for window.finalized < number {
		delete(window.blockNumber2Hashes, window.finalized)
		window.finalized++
		evicted++
	}

	return
}

func (window ReorgWindow) String() string {
	if len(window.blockNumber2Hashes) == 0 {
		return "{ blocks = 0 }"
	}

	return fmt.Sprintf("{ blocks = %v, finalized = %v, latest = %v }",
		len(window.blockNumber2Hashes), window.finalized, window.latest)
}
