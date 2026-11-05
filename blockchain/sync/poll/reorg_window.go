package poll

import "fmt"

type ReorgWindowParams struct {
	FinalizedBlockNumber uint64
	FinalizedBlockHash   string
	LatestBlocks         map[uint64]string
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

// NewReorgWindow creates an empty reorg window. Generally, it is used to sync from the genesis block.
func NewReorgWindow() *ReorgWindow {
	return &ReorgWindow{
		blockNumber2Hashes: make(map[uint64]string),
	}
}

// NewReorgWindowWithLatestBlocks creates a reorg window with given recent blocks and the last finalized block number.
//
// When service restarted, user should load recent blocks and the last finalized block from database,
// and initialize the reorg window so as to correctly handle chain reorg during the service down time.
func NewReorgWindowWithLatestBlocks(params ReorgWindowParams) (*ReorgWindow, error) {
	window := NewReorgWindow()

	if len(params.FinalizedBlockHash) == 0 {
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

		window.blockNumber2Hashes[expectedBlockNumber] = hash
		window.latest = expectedBlockNumber
	}

	return window, nil
}

// Push pushes the latest block into reorg window.
func (window *ReorgWindow) Push(blockNumber uint64, blockHash, parentBlockHash string) (appended, popped bool, err error) {
	// window is empty, e.g. sync from genesis block
	if len(window.blockNumber2Hashes) == 0 {
		window.finalized = blockNumber
		window.latest = blockNumber
		window.blockNumber2Hashes[blockNumber] = blockHash
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
