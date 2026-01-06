package core

import (
	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/DmitriyVTitov/size"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
)

type EpochData struct {
	Blocks   []*types.Block               // 1 pivot block at least
	Receipts [][]types.TransactionReceipt // nil if ignored
	Traces   *types.EpochTrace            // nil if ignored

	numTxs int // total number of transactions in epoch, including skipped ones
}

// Size implements the channel.Sizable interface.
func (data EpochData) Size() int {
	return size.Of(data)
}

func (data *EpochData) queryBlocks(client sdk.ClientOperator, epochNumber uint64) error {
	epoch := types.NewEpochNumberUint64(epochNumber)

	blockHashes, err := client.GetBlocksByEpoch(epoch)
	if err != nil {
		return errors.WithMessage(err, "Failed to get block hashes by epoch number")
	}

	numBlocks := len(blockHashes)
	if numBlocks == 0 {
		return errors.Errorf("Epoch blocks not found by number %v", epochNumber)
	}

	pivotHash := blockHashes[numBlocks-1]
	data.Blocks = make([]*types.Block, 0, numBlocks)

	for _, v := range blockHashes {
		block, err := client.GetBlockByHashWithPivotAssumption(v, pivotHash, hexutil.Uint64(epochNumber))
		if err != nil {
			return errors.WithMessage(err, "Failed to get block by hash with pivot assumption")
		}

		data.Blocks = append(data.Blocks, &block)
		data.numTxs += len(block.Transactions)
	}

	return nil
}

func (data *EpochData) queryReceipts(client sdk.ClientOperator) error {
	numBlocks := len(data.Blocks)

	// optimize for empty blocks
	if data.numTxs == 0 {
		data.Receipts = make([][]types.TransactionReceipt, 0, numBlocks)

		for range data.Blocks {
			data.Receipts = append(data.Receipts, []types.TransactionReceipt{})
		}

		return nil
	}

	// query receipts in batch
	pivotBlock := data.Blocks[numBlocks-1]
	epoch := types.NewEpochOrBlockHashWithBlockHash(pivotBlock.Hash, true)

	epochReceipts, err := client.GetEpochReceipts(*epoch)
	if err != nil {
		return errors.WithMessage(err, "Failed to get epoch receipts by pivot block hash")
	}

	// detect reorg
	epochNumber := pivotBlock.EpochNumber.ToInt().Uint64()
	for _, blockReceipts := range epochReceipts {
		for _, receipt := range blockReceipts {
			if receipt.EpochNumber == nil || uint64(*receipt.EpochNumber) != epochNumber {
				return errors.Errorf("Receipt epoch number mismatch %v", receipt.EpochNumber)
			}
		}
	}

	data.Receipts = epochReceipts

	return nil
}

func (data *EpochData) queryTraces(client sdk.ClientOperator) error {
	// optimize for empty blocks
	if data.numTxs == 0 {
		data.Traces = &types.EpochTrace{}
		return nil
	}

	pivotBlock := data.Blocks[len(data.Blocks)-1]
	epoch := types.NewEpochNumber(pivotBlock.EpochNumber)

	traces, err := client.Trace().GetEpochTraces(*epoch)
	if err != nil {
		return errors.WithMessage(err, "Failed to get epoch traces by epoch number")
	}

	// try to detect reorg
	for i, v := range traces.CfxTraces {
		if v.EpochHash == nil || *v.EpochHash != pivotBlock.Hash {
			return errors.Errorf("Trace epoch hash mismatch with pivot block hash, index = %v", i)
		}
	}

	data.Traces = &traces

	return nil
}
