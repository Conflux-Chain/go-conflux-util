package poll

import (
	"context"
	"sync"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/ctxutil"
	"github.com/Conflux-Chain/go-conflux-util/health"
	"github.com/pkg/errors"
)

type Option struct {
	IdleInterval  time.Duration `default:"1s"`
	RetryInterval time.Duration `default:"5s"`
	BufferSize    int           `default:"32"`
	Health        health.TimedCounterConfig
}

// FinalizedPoller is used to poll the finalized blockchain data block by block.
type FinalizedPoller[T any] struct {
	option          Option
	adapter         Adapter[T]
	nextBlockNumber uint64
	dataCh          chan T
	health          *health.TimedCounter
}

func NewFinalizedPoller[T any](adapter Adapter[T], nextBlockNumber uint64, option ...Option) *FinalizedPoller[T] {
	opt := normalizeOpt(option...)

	return &FinalizedPoller[T]{
		option:          opt,
		adapter:         adapter,
		nextBlockNumber: nextBlockNumber,
		dataCh:          make(chan T, opt.BufferSize),
		health:          health.NewTimedCounter(opt.Health),
	}
}

// DataCh returns a read-only channel to consume data. The channel will not be closed
// until poll goroutine terminated.
func (poller *FinalizedPoller[T]) DataCh() <-chan T {
	return poller.dataCh
}

// Poll polls the finalized blockchain data block by block.
func (poller *FinalizedPoller[T]) Poll(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	// close channel if completed
	defer close(poller.dataCh)

	for {
		data, ok, err := poller.pollOnce(ctx)

		// return if context done
		if ctxutil.IsDone(ctx) {
			return
		}

		poller.health.LogOnError(err, "Poll finalized blockchain data")

		if err != nil {
			err = ctxutil.Sleep(ctx, poller.option.RetryInterval)
		} else if ok {
			err = ctxutil.WriteChannel(ctx, poller.dataCh, data)
			poller.nextBlockNumber++
		} else {
			err = ctxutil.Sleep(ctx, poller.option.IdleInterval)
		}

		// context done
		if err != nil {
			return
		}
	}
}

func (poller *FinalizedPoller[T]) pollOnce(ctx context.Context) (data T, ok bool, err error) {
	// get the finalized block number
	finalizedBlockNumber, err := poller.adapter.GetFinalizedBlockNumber(ctx)
	if err != nil {
		return data, false, errors.WithMessage(err, "Failed to get finalized block number")
	}

	// already caught up
	if poller.nextBlockNumber > finalizedBlockNumber {
		return data, false, nil
	}

	// retrieve the next blockchain data
	if data, err = poller.adapter.GetBlockData(ctx, poller.nextBlockNumber); err != nil {
		return data, false, errors.WithMessage(err, "Failed to retrieve blockchain data")
	}

	return data, true, nil
}
