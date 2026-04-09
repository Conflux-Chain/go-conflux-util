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
type LatestPoller[T any] struct {
	option          Option
	adapter         Adapter[T]
	nextBlockNumber uint64
	dataCh          chan Revertable[T]
	window          *ReorgWindow
	health          *health.TimedCounter
}

func NewLatestPoller[T any](adapter Adapter[T], nextBlockNumber uint64, reorgParams ReorgWindowParams, option ...Option) (*LatestPoller[T], error) {
	opt := normalizeOpt(option...)

	window, err := NewReorgWindowWithLatestBlocks(reorgParams)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create reorg window")
	}

	return &LatestPoller[T]{
		option:          opt,
		adapter:         adapter,
		nextBlockNumber: nextBlockNumber,
		dataCh:          make(chan Revertable[T], opt.BufferSize),
		window:          window,
		health:          health.NewTimedCounter(opt.Health),
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
		data, ok, reorg, err := poller.pollOnce(ctx)

		// return if context done
		if ctxutil.IsDone(ctx) {
			return
		}

		poller.health.LogOnError(err, "Poll latest blockchain data")

		logger := log.WithModule(ModuleName).WithField("block", poller.nextBlockNumber)

		if err != nil {
			logger.WithError(err).Debug("Failed to poll latest data")
			err = ctxutil.Sleep(ctx, poller.option.RetryInterval)
		} else if ok {
			logger.Trace("Succeeded to poll latest data")
			err = ctxutil.WriteChannel(ctx, poller.dataCh, Revertable[T]{
				Data:     data,
				Reverted: reverted,
			})

			poller.nextBlockNumber++
			reverted = false
		} else if reorg {
			logger.Debug("Reorg detected")
			poller.nextBlockNumber--
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

func (poller *LatestPoller[T]) pollOnce(ctx context.Context) (data T, ok bool, reorg bool, err error) {
	// Evict finalized blocks in reorg window
	finalizedBlockNumber, err := poller.adapter.GetFinalizedBlockNumber(ctx)
	if err != nil {
		return data, false, false, errors.WithMessage(err, "Failed to get finalized block number")
	}

	poller.window.Evict(finalizedBlockNumber)

	// get the latest block number
	latestBlockNumber, err := poller.adapter.GetLatestBlockNumber(ctx)
	if err != nil {
		return data, false, false, errors.WithMessage(err, "Failed to get latest block number")
	}

	// already caught up
	if poller.nextBlockNumber > latestBlockNumber {
		return data, false, false, nil
	}

	// retrieve the next blockchain data
	if data, err = poller.adapter.GetBlockData(ctx, poller.nextBlockNumber); err != nil {
		return data, false, false, errors.WithMessage(err, "Failed to retrieve blockchain data")
	}

	// detect reorg
	blockHash := poller.adapter.GetBlockHash(data)
	parentBlockHash := poller.adapter.GetParentBlockHash(data)
	appended, popped, err := poller.window.Push(poller.nextBlockNumber, blockHash, parentBlockHash)

	// should never happen
	if err != nil {
		log.WithModule(ModuleName).WithError(err).WithFields(logrus.Fields{
			"block":  poller.nextBlockNumber,
			"hash":   blockHash,
			"parent": parentBlockHash,
			"window": poller.window,
		}).Fatal("Block not in sequence or finalized block reverted")
	}

	return data, appended, popped, nil
}
