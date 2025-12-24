package poll

import "context"

// Adapter adapts any data source to fetch blockchain data. Typically, a PRC adapter is used to
// fetch blockchain data from fullnode.
//
// Generic type T is used to support different kinds of blockchain data, e.g. Conflux core space and eSpace.
type Adapter[T any] interface {

	// GetFinalizedBlockNumber returns the finalized block number, which will no longer be reverted.
	GetFinalizedBlockNumber(ctx context.Context) (uint64, error)

	// GetLatestBlockNumber returns the latest block number, which may be reverted later.
	//
	// E.g. "safe", "latest" or "latest - N" block that not finalized yet.
	GetLatestBlockNumber(ctx context.Context) (uint64, error)

	// GetBlockData returns the whole blockchain data of the given block number. Typically, it includes
	// block, transactions, receipts and traces.
	//
	// The implementation should be responsible for chain reorg detection if the given block number is
	// not finalized yet.
	GetBlockData(ctx context.Context, blockNumber uint64) (T, error)

	// GetBlockHash returns the block hash of given blockchain data.
	GetBlockHash(data T) string
	// GetParentBlockHash returns the parent block hash of given blockchain data.
	GetParentBlockHash(data T) string
}
