package poll

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/ctxutil"
	"github.com/Conflux-Chain/go-conflux-util/health"
	"github.com/Conflux-Chain/go-conflux-util/parallel"
)

type ParallelOption struct {
	parallel.SerialOption

	RetryInterval time.Duration `default:"1s"`

	Health health.TimedCounterConfig
}

// ParallelWorker is used to poll blockchain data in parallel.
type ParallelWorker[T any] struct {
	option  ParallelOption
	adapter Adapter[T]
	offset  uint64 // block number offset to poll blockchain data
	dataCh  chan<- T
	polled  atomic.Uint64
	health  *health.TimedCounter
}

func NewParallelWorker[T any](adapter Adapter[T], offset uint64, dataCh chan<- T, option ...ParallelOption) *ParallelWorker[T] {
	opt := normalizeOpt(option...)

	return &ParallelWorker[T]{
		option:  opt,
		adapter: adapter,
		offset:  offset,
		dataCh:  dataCh,
		health:  health.NewTimedCounter(opt.Health),
	}
}

// ParallelDo implements the parallel.Interface[T] interface.
func (worker *ParallelWorker[T]) ParallelDo(ctx context.Context, routine, task int) (data T, err error) {
	bn := worker.offset + uint64(task)

	for {
		data, err = worker.adapter.GetBlockData(ctx, bn)

		worker.health.LogOnError(err, "Poll blockchain data in parallel")

		if err == nil {
			return data, nil
		}

		if err = ctxutil.Sleep(ctx, worker.option.RetryInterval); err != nil {
			return data, err
		}
	}
}

// ParallelCollect implements the parallel.Interface[T] interface.
func (worker *ParallelWorker[T]) ParallelCollect(ctx context.Context, result *parallel.Result[T]) error {
	if result.Err != nil {
		return result.Err
	}

	if err := ctxutil.WriteChannel(ctx, worker.dataCh, result.Value); err != nil {
		return err
	}

	worker.polled.Add(1)

	return nil
}

// Polled returns the number of actual polled data.
func (worker *ParallelWorker[T]) Polled() uint64 {
	return worker.polled.Load()
}
