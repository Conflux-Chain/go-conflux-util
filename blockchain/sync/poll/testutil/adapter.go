package testutil

import (
	"context"
	"fmt"

	"github.com/DmitriyVTitov/size"
)

type Data struct {
	Number     uint64
	Hash       string
	ParentHash string
}

func (data Data) Size() int {
	return size.Of(data)
}

type Adapter struct {
	finalized     []uint64
	nextFinalized int

	latestBlocks []Data
	nextLatest   int
}

func MustNewAdapter(finalized []uint64, latestBlocks []Data) *Adapter {
	if len(finalized) == 0 {
		panic("Finalized  value is empty")
	}

	for i := 1; i < len(finalized); i++ {
		if finalized[i-1] >= finalized[i] {
			panic("Invalid finalized range")
		}
	}

	return &Adapter{
		finalized:    finalized,
		latestBlocks: latestBlocks,
	}
}

func (adapter *Adapter) GetFinalizedBlockNumber(ctx context.Context) (uint64, error) {
	if adapter.nextFinalized == len(adapter.finalized) {
		return adapter.finalized[adapter.nextFinalized-1], nil
	}

	finalized := adapter.finalized[adapter.nextFinalized]

	adapter.nextFinalized++

	return finalized, nil
}

func (adapter *Adapter) GetLatestBlockNumber(ctx context.Context) (uint64, error) {
	if len(adapter.latestBlocks) == 0 {
		return adapter.finalized[len(adapter.finalized)-1], nil
	}

	return adapter.latestBlocks[len(adapter.latestBlocks)-1].Number, nil
}

func (adapter *Adapter) GetBlockData(ctx context.Context, blockNumber uint64) (Data, error) {
	if finalized := adapter.finalized[len(adapter.finalized)-1]; blockNumber <= finalized {
		data := Data{
			Number: blockNumber,
			Hash:   fmt.Sprintf("DataHash-%v", blockNumber),
		}

		if blockNumber > 0 {
			data.ParentHash = fmt.Sprintf("DataHash-%v", blockNumber-1)
		}

		return data, nil
	}

	data := adapter.latestBlocks[adapter.nextLatest]
	if data.Number != blockNumber {
		panic("Requested block number mismatch with expected latest blocks")
	}

	adapter.nextLatest++

	return data, nil
}

func (adapter *Adapter) GetBlockHash(data Data) string {
	return data.Hash
}

func (adapter *Adapter) GetParentBlockHash(data Data) string {
	return data.ParentHash
}
