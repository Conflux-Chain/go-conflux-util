package poll

// ReorgWindow is used to detect chain reorg.
type ReorgWindow struct {
	// no capacity, since latest_finalized is small enough
	blockNumber2Hashes map[uint64]string
	earlist, latest    uint64
}

func NewReorgWindow() *ReorgWindow {
	return &ReorgWindow{
		blockNumber2Hashes: make(map[uint64]string),
	}
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
