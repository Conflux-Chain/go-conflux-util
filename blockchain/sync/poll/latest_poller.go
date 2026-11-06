package poll

import (
	"context"
	"fmt"
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
// Normally, reorg detection uses the next block's parent hash and waits if that
// height is not yet available.
// At capacity, polling waits for finality to advance and checks the block after
// the anchor to free space or discard a fork before fetching the next block.
type LatestPoller[T any] struct {
	option  Option
	adapter Adapter[T]
	dataCh  chan Revertable[T]
	window  *ReorgWindow
	health  *health.TimedCounter

	// Cache finality observed before fetching block data, independently of polling progress.
	// The initial boundary is genesis (0), which never requires eviction.
	finalizedNumber uint64
	finalizedHash   string
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
	// If the reorg window is full, try to evict finalized blocks to make room for new ones.
	if poller.window.Full() {
		if err := poller.refreshFinalized(ctx); err != nil {
			return data, false, false, err
		}

		anchor := poller.window.FinalizedBlockNumber()
		if poller.finalizedNumber <= anchor {
			return data, false, false, fmt.Errorf("Finalized block number not advanced, anchor = %v, finalized = %v", anchor, poller.finalizedNumber)
		}

		// Finality was observed before this query, so anchor+1 is immutable.
		confirmed, err := poller.adapter.GetBlockData(ctx, anchor+1)
		if err != nil {
			return data, false, false, errors.WithMessage(err, "Failed to retrieve block after finalized anchor")
		}

		if poller.window.Evict(anchor+1, poller.adapter.GetBlockHash(confirmed)) == 0 {
			// The first block after the anchor is on another fork, as are its descendants.
			poller.window.RewindToAnchor()
			return data, false, true, nil
		}

		// one room available after eviction
		return data, false, false, nil
	}

	// get the latest block number
	latestBlockNumber, err := poller.adapter.GetLatestBlockNumber(ctx)
	if err != nil {
		return data, false, false, errors.WithMessage(err, "Failed to get latest block number")
	}

	// already caught up
	if blockNumber > latestBlockNumber {
		return data, false, false, nil
	}

	// Reuse the finalized boundary while replaying history. Query before fetching
	// data so blocks inside that boundary are already immutable when retrieved.
	if blockNumber > poller.finalizedNumber {
		if err := poller.refreshFinalized(ctx); err != nil {
			// Finality maintenance is optional; retain any previously known boundary.
			log.WithModule(ModuleName).WithError(err).Debug("Failed to get finalized block for reorg window eviction")
		}
	}

	// retrieve the next blockchain data
	if data, err = poller.adapter.GetBlockData(ctx, blockNumber); err != nil {
		return data, false, false, errors.WithMessage(err, "Failed to retrieve blockchain data")
	}

	// detect reorg
	blockHash := poller.adapter.GetBlockHash(data)
	parentBlockHash := poller.adapter.GetParentBlockHash(data)
	appended, popped, err := poller.window.Push(blockNumber, blockHash, parentBlockHash)
	if errors.Is(err, ErrReorgWindowFull) {
		return data, false, false, err
	}

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
		if blockNumber < poller.finalizedNumber {
			// This block was already finalized when fetched. Keep it as the new
			// anchor and discard earlier hashes now that Push has accepted it.
			poller.window.Evict(blockNumber, blockHash)
		} else {
			// At or beyond the boundary, match the hash from the finality query.
			poller.window.Evict(poller.finalizedNumber, poller.finalizedHash)
		}
	}

	return data, appended, popped, nil
}

func (poller *LatestPoller[T]) refreshFinalized(ctx context.Context) error {
	number, hash, err := poller.adapter.GetFinalizedBlock(ctx)
	if err != nil {
		return errors.WithMessage(err, "Failed to get finalized block")
	}

	if number > poller.finalizedNumber {
		poller.finalizedNumber = number
		poller.finalizedHash = hash
	}

	return nil
}
