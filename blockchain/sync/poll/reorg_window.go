package poll

type ReorgWindowParams struct {
	FinalizedBlockNumber uint64
	FinalizedBlockHash   string
	LatestBlocks         map[uint64]string
}

// ReorgWindow is used to detect chain reorg.
type ReorgWindow struct {
	// no capacity, since max(latest_finalized, latest_checkpoint) is small enough
	blockNumber2Hashes map[uint64]string
	earlist, latest    uint64
}

func NewReorgWindow() *ReorgWindow {
	return &ReorgWindow{
		blockNumber2Hashes: make(map[uint64]string),
	}
}

// NewReorgWindowWithLatestBlocks initializes with given latest blocks and the last finalized block number.
//
// When service restarted, user should load recent blocks and the last finalized block from database,
// and initialize the reorg window so as to correctly handle chain reorg during the service down time.
func NewReorgWindowWithLatestBlocks(params ReorgWindowParams) *ReorgWindow {
	window := NewReorgWindow()
	if len(params.FinalizedBlockHash) == 0 {
		return window
	}

	window.earlist = params.FinalizedBlockNumber
	window.latest = params.FinalizedBlockNumber
	window.blockNumber2Hashes[params.FinalizedBlockNumber] = params.FinalizedBlockHash

	for {
		hash, ok := params.LatestBlocks[window.latest+1]
		if !ok {
			break
		}

		window.latest++
		window.blockNumber2Hashes[window.latest] = hash
	}

	return window
}

func (window *ReorgWindow) Push(blockNumber uint64, blockHash, parentBlockHash string) (appended, popped bool) {
	// window is empty
	if len(window.blockNumber2Hashes) == 0 {
		window.earlist = blockNumber
		window.latest = blockNumber
		window.blockNumber2Hashes[blockNumber] = blockHash
		return true, false
	}

	// not in sequence
	if window.latest+1 != blockNumber {
		return false, false
	}

	// appended
	if window.blockNumber2Hashes[window.latest] == parentBlockHash {
		window.blockNumber2Hashes[blockNumber] = blockHash
		window.latest = blockNumber
		return true, false
	}

	// parent block hash mismatch, reorg detected
	delete(window.blockNumber2Hashes, window.latest)
	window.latest--

	return false, true
}

func (window *ReorgWindow) Evict(blockNumber uint64) {
	if len(window.blockNumber2Hashes) == 0 {
		return
	}

	for i, end := window.earlist, min(blockNumber, window.latest); i <= end; i++ {
		delete(window.blockNumber2Hashes, i)
		window.earlist++
	}
}
