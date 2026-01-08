package sync

import (
	"context"
	"sync"

	"github.com/Conflux-Chain/go-conflux-util/blockchain/sync/poll"
	"github.com/Conflux-Chain/go-conflux-util/blockchain/sync/process"
	"github.com/Conflux-Chain/go-conflux-util/blockchain/sync/process/db"
	"github.com/Conflux-Chain/go-conflux-util/channel"
	"gorm.io/gorm"
)

type CatchupParamsDB[T any] struct {
	Adapter         poll.Adapter[T]
	Poller          poll.CatchUpOption
	Processor       db.BatchOption
	DB              *gorm.DB
	NextBlockNumber uint64
}

type ParamsDB[T any] struct {
	Adapter         poll.Adapter[T]
	Poller          poll.Option
	Processor       db.Option
	DB              *gorm.DB
	NextBlockNumber uint64

	// only used to sync latest data, and usually loads from database
	Reorg poll.ReorgWindowParams
}

func CatchUpDB[T channel.Sizable](ctx context.Context, params CatchupParamsDB[T], processors ...db.BatchProcessor[T]) uint64 {
	var wg sync.WaitGroup

	poller := poll.NewCatchUpPoller(params.Adapter, params.NextBlockNumber, params.Poller)
	wg.Add(1)
	go poller.Poll(ctx, &wg)

	processor := db.NewBatchAggregateProcessor(params.Processor, params.DB, processors...)
	wg.Add(1)
	go process.Process(ctx, &wg, poller.DataCh(), processor)

	wg.Wait()

	return poller.NextBlockNumber()
}

func StartFinalizedDB[T any](ctx context.Context, wg *sync.WaitGroup, params ParamsDB[T], processors ...db.Processor[T]) {
	poller := poll.NewFinalizedPoller(params.Adapter, params.NextBlockNumber, params.Poller)
	wg.Add(1)
	go poller.Poll(ctx, wg)

	processor := db.NewAggregateProcessor(params.Processor, params.DB, processors...)
	wg.Add(1)
	go process.Process(ctx, wg, poller.DataCh(), processor)
}

func StartLatestDB[T any](ctx context.Context, wg *sync.WaitGroup, params ParamsDB[T], processors ...db.RevertableProcessor[T]) {
	poller := poll.NewLatestPoller(params.Adapter, params.NextBlockNumber, params.Reorg, params.Poller)
	wg.Add(1)
	go poller.Poll(ctx, wg)

	processor := db.NewRevertableAggregateProcessor(params.Processor, params.DB, processors...)
	wg.Add(1)
	go process.Process(ctx, wg, poller.DataCh(), processor)
}
