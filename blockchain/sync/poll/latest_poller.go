package poll

import (
	"context"
	"sync"

	"github.com/Conflux-Chain/go-conflux-util/ctxutil"
	"github.com/Conflux-Chain/go-conflux-util/health"
	"github.com/Conflux-Chain/go-conflux-util/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Revertable[T any] struct {
	Data     T
	Reverted bool // indicates whether chain reorg happened
}

// LatestPoller is used to poll the latest blockchain data block by block.
// Reorg detection is driven by the next block's parent hash. If the chain head
// stays below the next height to poll, corrections wait until polling can resume.
type LatestPoller[T any] struct {
	option  Option
	adapter Adapter[T]
	dataCh  chan Revertable[T]
	window  *ReorgWindow
	health  *health.TimedCounter
}

// NewLatestPoller creates a poller starting from genesis if reorgParams is empty.
// With a restored window, polling starts after its highest block (the finalized
// anchor if no later blocks are supplied).
func NewLatestPoller[T any](adapter Adapter[T], reorgParams ReorgWindowParams, option ...Option) (*LatestPoller[T], error) {
	opt := normalizeOpt(option...)

	window, err := NewReorgWindow(reorgParams)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create reorg window")
	}

	return &LatestPoller[T]{
		option:  opt,
		adapter: adapter,
		dataCh:  make(chan Revertable[T], opt.BufferSize),
		window:  window,
		health:  health.NewTimedCounter(opt.Health),
	}, nil
}

// DataCh returns a read-only channel to consume data. The channel will not be closed
// until poll goroutine terminated.
func (poller *LatestPoller[T]) DataCh() <-chan Revertable[T] {
	return poller.dataCh
}

// Poll polls the latest blockchain data block by block.
func (poller *LatestPoller[T]) Poll(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	// close channel if completed
	defer close(poller.dataCh)

	var reverted bool

	for {
		blockNumber := poller.window.NextBlockNumber()
		data, ok, reorg, err := poller.pollOnce(ctx, blockNumber)

		// return if context done
		if ctxutil.IsDone(ctx) {
			return
		}

		poller.health.LogOnError(err, "Poll latest blockchain data")

		logger := log.WithModule(ModuleName).WithField("block", blockNumber)

		if err != nil {
			logger.WithError(err).Debug("Failed to poll latest data")
			err = ctxutil.Sleep(ctx, poller.option.RetryInterval)
		} else if ok {
			logger.Trace("Succeeded to poll latest data")
			err = ctxutil.WriteChannel(ctx, poller.dataCh, Revertable[T]{
				Data:     data,
				Reverted: reverted,
			})

			reverted = false
		} else if reorg {
			logger.Debug("Reorg detected")
			reverted = true
		} else {
			logger.Trace("No latest data to poll")
			err = ctxutil.Sleep(ctx, poller.option.IdleInterval)
		}

		// context done
		if err != nil {
			return
		}
	}
}

func (poller *LatestPoller[T]) pollOnce(ctx context.Context, blockNumber uint64) (data T, ok bool, reorg bool, err error) {
	// get the latest block number
	latestBlockNumber, err := poller.adapter.GetLatestBlockNumber(ctx)
	if err != nil {
		return data, false, false, errors.WithMessage(err, "Failed to get latest block number")
	}

	// already caught up
	if blockNumber > latestBlockNumber {
		return data, false, false, nil
	}

	// retrieve the next blockchain data
	if data, err = poller.adapter.GetBlockData(ctx, blockNumber); err != nil {
		return data, false, false, errors.WithMessage(err, "Failed to retrieve blockchain data")
	}

	// detect reorg
	blockHash := poller.adapter.GetBlockHash(data)
	parentBlockHash := poller.adapter.GetParentBlockHash(data)
	appended, popped, err := poller.window.Push(blockNumber, blockHash, parentBlockHash)

	// should never happen
	if err != nil {
		log.WithModule(ModuleName).WithError(err).WithFields(logrus.Fields{
			"block":  blockNumber,
			"hash":   blockHash,
			"parent": parentBlockHash,
			"window": poller.window,
		}).Fatal("Block not in sequence or finalized block reverted")
	}

	if appended {
		// Eviction is optional maintenance. A failed finality query must not
		// prevent delivery of a block already appended to the window.
		number, hash, err := poller.adapter.GetFinalizedBlock(ctx)
		if err != nil {
			log.WithModule(ModuleName).WithError(err).Debug("Failed to get finalized block for reorg window eviction")
		} else {
			poller.window.Evict(number, hash)
		}
	}

	return data, appended, popped, nil
}
