package evm

import (
	"context"
	"fmt"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/blockchain/sync/poll"
	"github.com/mcuadros/go-defaults"
	providers "github.com/openweb3/go-rpc-provider/provider_wrapper"
	"github.com/openweb3/web3go"
	"github.com/openweb3/web3go/types"
	"github.com/pkg/errors"
)

type AdapterConfig struct {
	URL string

	Option AdapterOption
}

type AdapterOption struct {
	// RPC
	RequestTimeout time.Duration `default:"3s"`

	// latest block number
	LatestBlockNumberTag    int64  `default:"-1"` // "latest" block
	LatestBlockNumberOffset uint64 `default:"5"`

	// allow to ignore receipts and/or traces, only block and transactions are required
	IgnoreReceipts bool
	IgnoreTraces   bool
}

var _ poll.Adapter[BlockData] = (*Adapter)(nil)

// Adapter implements the poll.Adapter[T] interface to poll data from evm RPC.
type Adapter struct {
	option AdapterOption

	client *web3go.Client
}

func NewAdapter(url string, option AdapterOption) (*Adapter, error) {
	defaults.SetDefaults(&option)

	clientOption := web3go.ClientOption{
		Option: providers.Option{
			RequestTimeout: option.RequestTimeout,
		},
	}

	client, err := web3go.NewClientWithOption(url, clientOption)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create client")
	}

	return &Adapter{option, client}, nil
}

func NewAdapterWithConfig(config AdapterConfig) (*Adapter, error) {
	if len(config.URL) == 0 {
		return nil, errors.New("URL not specified")
	}

	return NewAdapter(config.URL, config.Option)
}

// Close closes the underlying RPC client.
func (adapter *Adapter) Close() {
	adapter.client.Close()
}

// GetFinalizedBlockNumber implements the poll.Adapter[T] interface.
func (adapter *Adapter) GetFinalizedBlockNumber(ctx context.Context) (uint64, error) {
	block, err := adapter.client.WithContext(ctx).Eth.BlockByNumber(types.FinalizedBlockNumber, false)
	if err != nil {
		return 0, err
	}

	return block.Number.Uint64(), nil
}

// GetLatestBlockNumber implements the poll.Adapter[T] interface.
func (adapter *Adapter) GetLatestBlockNumber(ctx context.Context) (uint64, error) {
	block, err := adapter.client.WithContext(ctx).Eth.BlockByNumber(types.BlockNumber(adapter.option.LatestBlockNumberTag), false)
	if err != nil {
		return 0, err
	}

	bn := block.Number.Uint64()
	if bn < adapter.option.LatestBlockNumberOffset {
		return 0, nil
	}

	return bn - adapter.option.LatestBlockNumberOffset, nil
}

// GetBlockData implements the poll.Adapter[T] interface.
func (adapter *Adapter) GetBlockData(ctx context.Context, blockNumber uint64) (BlockData, error) {
	var data BlockData

	bn := types.BlockNumber(blockNumber)

	if err := data.queryBlock(adapter.client, bn); err != nil {
		return BlockData{}, errors.WithMessage(err, "Failed to query block")
	}

	if !adapter.option.IgnoreReceipts {
		if err := data.queryReceipts(adapter.client, bn); err != nil {
			return BlockData{}, errors.WithMessage(err, "Failed to query receipts")
		}
	}

	if !adapter.option.IgnoreTraces {
		if err := data.queryTraces(adapter.client, bn); err != nil {
			return BlockData{}, errors.WithMessage(err, "Failed to query traces")
		}
	}

	return data, nil
}

// GetBlockHash implements the poll.Adapter[T] interface.
func (adapter *Adapter) GetBlockHash(data BlockData) string {
	return data.Block.Hash.Hex()
}

// GetParentBlockHash implements the poll.Adapter[T] interface.
func (adapter *Adapter) GetParentBlockHash(data BlockData) string {
	return data.Block.ParentHash.Hex()
}
