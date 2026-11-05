package sync

import (
	"context"
	"sync"

	"github.com/Conflux-Chain/go-conflux-util/blockchain/sync/poll"
	"github.com/Conflux-Chain/go-conflux-util/blockchain/sync/process"
	"github.com/Conflux-Chain/go-conflux-util/blockchain/sync/process/db"
	"github.com/Conflux-Chain/go-conflux-util/channel"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type CatchupParamsDB[T any] struct {
	Adapter         poll.Adapter[T]
	Poller          poll.CatchUpOption
	Processor       db.BatchOption
	DB              *gorm.DB
	NextBlockNumber uint64
}

type FinalizedParamsDB[T any] struct {
	Adapter         poll.Adapter[T]
	Poller          poll.Option
	Processor       db.Option
	DB              *gorm.DB
	NextBlockNumber uint64
}

type LatestParamsDB[T any] struct {
	Adapter   poll.Adapter[T]
	Poller    poll.Option
	Processor db.Option
	DB        *gorm.DB
	Reorg     poll.ReorgWindowParams
}

// CatchUpDB polls finalized blocks in parallel from params.NextBlockNumber and
// processes them in order in database batches. It blocks until polling and processing stop.
// On normal completion, all polled data and the final batch have been processed,
// and the returned height is the next block to sync.
//
// Cancellation stops synchronization without guaranteeing that buffered data or
// the final batch is committed. If ctx is canceled, ignore the returned height
// and resume from the database's committed progress on the next run.
// Any remaining channel data is discarded before returning.
func CatchUpDB[T channel.Sizable](ctx context.Context, params CatchupParamsDB[T], processors ...db.BatchProcessor[T]) uint64 {
	var wg sync.WaitGroup

	poller := poll.NewCatchUpPoller(params.Adapter, params.NextBlockNumber, params.Poller)
	wg.Add(1)
	go poller.Poll(ctx, &wg)

	processor := db.NewBatchAggregateProcessor(params.Processor, params.DB, processors...)
	wg.Add(1)
	go process.ProcessCatchUp(ctx, &wg, poller.DataCh(), processor)

	wg.Wait()

	// Discard remaining buffered data so the forwarding goroutine can exit.
	for range poller.DataCh() {
	}

	return poller.NextBlockNumber()
}

// StartFinalizedDB starts continuous synchronization of finalized blocks from
// params.NextBlockNumber, polling and processing blocks sequentially in separate
// goroutines. It returns immediately and keeps polling for new finalized blocks
// until ctx is canceled.
//
// The polling and processing goroutines are registered with wg. After canceling
// ctx, wait on wg before closing the database. Cancellation does not guarantee
// that buffered data is committed; resume from committed database progress.
func StartFinalizedDB[T any](ctx context.Context, wg *sync.WaitGroup, params FinalizedParamsDB[T], processors ...db.Processor[T]) {
	poller := poll.NewFinalizedPoller(params.Adapter, params.NextBlockNumber, params.Poller)
	wg.Add(1)
	go poller.Poll(ctx, wg)

	processor := db.NewAggregateProcessor(params.Processor, params.DB, processors...)
	wg.Add(1)
	go process.Process(ctx, wg, poller.DataCh(), processor)
}

// StartLatestDB starts continuous synchronization of latest blocks,
// polling and processing blocks sequentially in separate
// goroutines and using revertable processors to handle chain reorganizations.
// params.Reorg supplies the finalized anchor and recent block hashes needed to
// restore reorg tracking when resuming from the database. Polling starts after
// that window's highest block, or from genesis if Reorg is empty.
//
// It returns after starting the goroutines, or returns an initialization error
// without starting them. The goroutines are registered with wg and run until ctx
// is canceled. Wait on wg after cancellation before closing the database.
// Buffered data is not guaranteed to be committed on cancellation; resume from
// committed database progress and restore the corresponding reorg state.
func StartLatestDB[T any](ctx context.Context, wg *sync.WaitGroup, params LatestParamsDB[T], processors ...db.RevertableProcessor[T]) error {
	poller, err := poll.NewLatestPoller(params.Adapter, params.Reorg, params.Poller)
	if err != nil {
		return errors.WithMessage(err, "Failed to create latest poller")
	}

	wg.Add(1)
	go poller.Poll(ctx, wg)

	processor := db.NewRevertableAggregateProcessor(params.Processor, params.DB, processors...)
	wg.Add(1)
	go process.Process(ctx, wg, poller.DataCh(), processor)

	return nil
}
