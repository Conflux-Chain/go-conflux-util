package poll

import (
	"context"
	"sync"

	"github.com/Conflux-Chain/go-conflux-util/channel"
	"github.com/Conflux-Chain/go-conflux-util/ctxutil"
	"github.com/Conflux-Chain/go-conflux-util/health"
	"github.com/Conflux-Chain/go-conflux-util/parallel"
	"github.com/mcuadros/go-defaults"
	"github.com/pkg/errors"
)

type CatchUpOption struct {
	Parallel ParallelOption

	Buffer struct {
		Capacity int `default:"1024"`
		MaxBytes int `default:"256000000"` // 256M
	}
}

// CatchUpPoller is used to poll blockchain data concurrently in catch up phase.
type CatchUpPoller[T channel.Sizable] struct {
	option          CatchUpOption
	adapter         Adapter[T]
	nextBlockNumber uint64
	dataCh          *channel.MemoryBoundedChannel[T] // must bounds the memory to avoid OOM
	health          *health.TimedCounter
}

func normalizeOpt[T any](option ...T) T {
	var opt T

	if len(option) > 0 {
		opt = option[0]
	}

	defaults.SetDefaults(&opt)

	return opt
}

func NewCatchUpPoller[T channel.Sizable](adapter Adapter[T], nextBlockNumber uint64, option ...CatchUpOption) *CatchUpPoller[T] {
	opt := normalizeOpt(option...)

	return &CatchUpPoller[T]{
		option:          opt,
		adapter:         adapter,
		nextBlockNumber: nextBlockNumber,
		dataCh:          channel.NewMemoryBoundedChannel[T](opt.Buffer.Capacity, opt.Buffer.MaxBytes),
		health:          health.NewTimedCounter(opt.Parallel.Health),
	}
}

// DataCh returns a read-only channel to consume data.
//
// Note, the channel wil be closed once caught up to the latest finalized block.
func (poller *CatchUpPoller[T]) DataCh() <-chan T {
	return poller.dataCh.RecvCh()
}

// NextBlockNumber returns the next block number to poll data.
func (poller *CatchUpPoller[T]) NextBlockNumber() uint64 {
	return poller.nextBlockNumber
}

// Poll polls blockchain data in parallel until the latest finalized block number is polled.
func (poller *CatchUpPoller[T]) Poll(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	// close channel if catch up completed
	defer poller.dataCh.Close()

	for {
		blocks, err := poller.pollOnce(ctx)

		// return if context done
		if ctxutil.IsDone(ctx) {
			return
		}

		poller.health.LogOnError(err, "Poll blockchain data in catch up mode")

		// catch up once again if any blocks polled
		if blocks > 0 {
			continue
		}

		// already caught up
		if err == nil {
			return
		}

		// retry
		if err = ctxutil.Sleep(ctx, poller.option.Parallel.RetryInterval); err != nil {
			return
		}
	}
}

func (poller *CatchUpPoller[T]) pollOnce(ctx context.Context) (int, error) {
	// get the finalized block number
	finalizedBlockNumber, err := poller.adapter.GetFinalizedBlockNumber(ctx)
	if err != nil {
		return 0, errors.WithMessage(err, "Failed to get finalized block number")
	}

	// already caught up
	if poller.nextBlockNumber > finalizedBlockNumber {
		return 0, nil
	}

	// poll data in parallel
	worker := NewParallelWorker(poller.adapter, poller.nextBlockNumber, poller.dataCh.SendCh(), poller.option.Parallel)
	tasks := int(finalizedBlockNumber - poller.nextBlockNumber + 1)
	err = parallel.Serial(ctx, worker, tasks, poller.option.Parallel.SerialOption)
	poller.nextBlockNumber += worker.Polled()
	if err != nil {
		return 0, errors.WithMessage(err, "Failed to poll blockchain data in parallel")
	}

	return tasks, nil
}
