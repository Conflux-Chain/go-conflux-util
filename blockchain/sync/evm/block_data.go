package evm

import (
	"github.com/DmitriyVTitov/size"
	"github.com/openweb3/web3go"
	"github.com/openweb3/web3go/types"
	"github.com/pkg/errors"
)

type BlockData struct {
	Block    *types.Block           // always not nil
	Receipts []*types.Receipt       // nil if ignored, empty slice if no tx in block
	Traces   []types.LocalizedTrace // nil if ignored, empty slice if no tx in block
}

// Size implements the channel.Sizable interface.
func (data BlockData) Size() int {
	return size.Of(data)
}

func (data *BlockData) queryBlock(client *web3go.Client, blockNumber types.BlockNumber) error {
	block, err := client.Eth.BlockByNumber(blockNumber, true)
	if err != nil {
		return errors.WithMessage(err, "Failed to get block by number")
	}

	if block == nil {
		return errors.Errorf("Block not found by number %v", blockNumber.Int64())
	}

	data.Block = block

	return nil
}

func (data *BlockData) queryReceipts(client *web3go.Client, blockNumber types.BlockNumber) error {
	txs := data.Block.Transactions.Transactions()
	if len(txs) == 0 {
		data.Receipts = []*types.Receipt{}
		return nil
	}

	bnoh := types.BlockNumberOrHashWithNumber(blockNumber)
	receipts, err := client.Eth.BlockReceipts(&bnoh)
	if err != nil {
		return errors.WithMessage(err, "Failed to get block receipts by block number")
	}

	if receipts == nil {
		return errors.Errorf("Receipts not found by block %v", blockNumber.Int64())
	}

	// detect temp chain reorg
	if len(receipts) != len(txs) {
		return errors.Errorf("Receipts length and txs length mismatch, receipts = %v, txs = %v", len(receipts), len(txs))
	}

	for i, v := range receipts {
		if v.BlockHash != data.Block.Hash {
			return errors.Errorf("Receipt block hash mismatch, index = %v", i)
		}

		if v.TransactionHash != txs[i].Hash {
			return errors.Errorf("Receipt tx hash mismatch, index = %v", i)
		}
	}

	data.Receipts = receipts

	return nil
}

func (data *BlockData) queryTraces(client *web3go.Client, blockNumber types.BlockNumber) error {
	txs := data.Block.Transactions.Transactions()
	if len(txs) == 0 {
		data.Traces = []types.LocalizedTrace{}
		return nil
	}

	bnoh := types.BlockNumberOrHashWithNumber(blockNumber)
	traces, err := client.Trace.Blocks(bnoh)
	if err != nil {
		return errors.WithMessage(err, "Failed to get block traces by block number")
	}

	if traces == nil {
		return errors.Errorf("Traces not found by block %v", blockNumber.Int64())
	}

	// Try to detect temp chain reorg if there is any trace.
	// Otherwise, temp chain reorg may lead to data inconsistency issue.
	for i, v := range traces {
		if v.BlockHash != data.Block.Hash {
			return errors.Errorf("Trace block hash mismatch, index = %v", i)
		}
	}

	data.Traces = traces

	return nil
}
