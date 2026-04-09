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
// Note, the reorg window do not require a capacity. Because, the data size of block number
// and hash are very small. Besides, there are a few blocks between the finalized block and latest block.
type ReorgWindow struct {
	blockNumber2Hashes map[uint64]string // recent blocks in sequence
	finalized          uint64            // the finalized block number, which will not be reverted
	latest             uint64            // the latest block number, which may be reverted
	verified           bool              // whether verified by on-chain data
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
		window.verified = true
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
		window.verified = true
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

// Evict evicts finalized blocks from reorg window.
func (window *ReorgWindow) Evict(finalizedBlockNumber uint64) (evicted int) {
	if len(window.blockNumber2Hashes) == 0 {
		return
	}

	// When service restarted, the reorg window will be initialized with recent blocks from database.
	// These blocks may be reverted during service down time, and requires to compare with on-chain blocks.
	// So, only when new on-chain block successfully pushed into the reorg window, the recent blocks could
	// be evicted from reorg window.
	if !window.verified {
		return
	}

	// keep at latest one finalized block
	evictUntil := min(finalizedBlockNumber, window.latest)

	for i := window.finalized; i < evictUntil; i++ {
		delete(window.blockNumber2Hashes, i)
		window.finalized++
		evicted++
	}

	return
}

func (window ReorgWindow) String() string {
	if len(window.blockNumber2Hashes) == 0 {
		return "{ blocks = 0 }"
	}

	return fmt.Sprintf("{ blocks = %v, finalized = %v, latest = %v, verified = %v }",
		len(window.blockNumber2Hashes), window.finalized, window.latest, window.verified)
}
