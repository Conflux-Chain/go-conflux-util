package core

import (
	"context"
	"time"

	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/Conflux-Chain/go-conflux-util/blockchain/sync/poll"
	"github.com/mcuadros/go-defaults"
	"github.com/pkg/errors"
)

const defaultRpcRetry = 3

type AdapterConfig struct {
	URL string

	Option AdapterOption
}

type AdapterOption struct {
	// RPC
	RequestTimeout time.Duration `default:"3s"`

	// latest block number
	LatestBlockNumberTag    string `default:"latest_state"`
	LatestBlockNumberOffset uint64 `default:"5"`
	latestEpoch             *types.Epoch

	// allow to ignore receipts and/or traces, only block and transactions are required
	IgnoreReceipts bool
	IgnoreTraces   bool
}

var _ poll.Adapter[EpochData] = (*Adapter)(nil)

// Adapter implements the poll.Adapter[T] interface to poll data from evm RPC.
type Adapter struct {
	option AdapterOption

	client *sdk.Client
}

func NewAdapter(url string, option AdapterOption) (*Adapter, error) {
	defaults.SetDefaults(&option)

	switch option.LatestBlockNumberTag {
	case types.EpochLatestMined.String():
		option.latestEpoch = types.EpochLatestMined
	case types.EpochLatestState.String():
		option.latestEpoch = types.EpochLatestState
	case types.EpochLatestConfirmed.String():
		option.latestEpoch = types.EpochLatestConfirmed
	default:
		return nil, errors.Errorf("Invalid latest block number: %v", option.LatestBlockNumberTag)
	}

	clientOption := sdk.ClientOption{
		RequestTimeout: option.RequestTimeout,
	}

	client, err := sdk.NewClient(url, clientOption)
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
	var retry int

	// retry for rpc failure due to temp chain reorg
	for {
		status, err := adapter.client.WithContext(ctx).GetStatus()
		if err == nil {
			return max(uint64(status.LatestCheckpoint), uint64(status.LatestFinalized)), nil
		}

		retry++

		if retry > defaultRpcRetry {
			return 0, err
		}
	}
}

// GetLatestBlockNumber implements the poll.Adapter[T] interface.
func (adapter *Adapter) GetLatestBlockNumber(ctx context.Context) (uint64, error) {
	number, err := adapter.client.WithContext(ctx).GetEpochNumber(adapter.option.latestEpoch)
	if err != nil {
		return 0, err
	}

	numberU64 := number.ToInt().Uint64()

	if numberU64 < adapter.option.LatestBlockNumberOffset {
		return 0, nil
	}

	return numberU64 - adapter.option.LatestBlockNumberOffset, nil
}

// GetBlockData implements the poll.Adapter[T] interface.
func (adapter *Adapter) GetBlockData(ctx context.Context, blockNumber uint64) (EpochData, error) {
	var data EpochData

	if err := data.queryBlocks(adapter.client, blockNumber); err != nil {
		return EpochData{}, errors.WithMessage(err, "Failed to query epoch blocks")
	}

	if !adapter.option.IgnoreReceipts {
		if err := data.queryReceipts(adapter.client); err != nil {
			return EpochData{}, errors.WithMessage(err, "Failed to query epoch receipts")
		}
	}

	if !adapter.option.IgnoreTraces {
		if err := data.queryTraces(adapter.client); err != nil {
			return EpochData{}, errors.WithMessage(err, "Failed to query epoch traces")
		}
	}

	return data, nil
}

// GetBlockHash implements the poll.Adapter[T] interface.
func (adapter *Adapter) GetBlockHash(data EpochData) string {
	return data.Blocks[len(data.Blocks)-1].Hash.String()
}

// GetParentBlockHash implements the poll.Adapter[T] interface.
func (adapter *Adapter) GetParentBlockHash(data EpochData) string {
	return data.Blocks[len(data.Blocks)-1].ParentHash.String()
}
