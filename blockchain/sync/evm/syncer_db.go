package evm

import (
	"context"
	"sync"

	"github.com/Conflux-Chain/go-conflux-util/blockchain/sync/poll"
	"github.com/Conflux-Chain/go-conflux-util/blockchain/sync/process"
	"github.com/Conflux-Chain/go-conflux-util/blockchain/sync/process/db"
	"github.com/mcuadros/go-defaults"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type CatchUpConfig struct {
	Adapter   AdapterConfig
	Poller    poll.CatchUpOption
	Processor db.BatchOption
}

type Config struct {
	Adapter   AdapterConfig
	Poller    poll.Option
	Processor db.Option
}

func CatchUpDB(ctx context.Context, config CatchUpConfig, DB *gorm.DB, nextBlockNumber uint64, processors ...db.BatchProcessor[BlockData]) (uint64, error) {
	defaults.SetDefaults(&config)

	adapter, err := NewAdapter(config.Adapter)
	if err != nil {
		return 0, errors.WithMessage(err, "Failed to create adapter")
	}

	var wg sync.WaitGroup

	poller := poll.NewCatchUpPoller(adapter, nextBlockNumber, config.Poller)
	wg.Add(1)
	go poller.Poll(ctx, &wg)

	processor := db.NewBatchAggregateProcessor(config.Processor, DB, processors...)
	wg.Add(1)
	go process.Process(ctx, &wg, poller.DataCh(), processor)

	wg.Wait()

	return poller.NextBlockNumber(), nil
}

func StartFinalizedDB(ctx context.Context, wg *sync.WaitGroup, config Config, DB *gorm.DB, nextBlockNumber uint64, processors ...db.Processor[BlockData]) error {
	defaults.SetDefaults(&config)

	adapter, err := NewAdapter(config.Adapter)
	if err != nil {
		return errors.WithMessage(err, "Failed to create adapter")
	}

	poller := poll.NewFinalizedPoller(adapter, nextBlockNumber, config.Poller)
	wg.Add(1)
	go poller.Poll(ctx, wg)

	processor := db.NewAggregateProcessor(config.Processor, DB, processors...)
	wg.Add(1)
	go process.Process(ctx, wg, poller.DataCh(), processor)

	return nil
}

func StartLatestDB(ctx context.Context, wg *sync.WaitGroup, config Config, DB *gorm.DB, nextBlockNumber uint64, processors ...db.RevertableProcessor[BlockData]) error {
	defaults.SetDefaults(&config)

	adapter, err := NewAdapter(config.Adapter)
	if err != nil {
		return errors.WithMessage(err, "Failed to create adapter")
	}

	poller := poll.NewLatestPoller(adapter, nextBlockNumber, config.Poller)
	wg.Add(1)
	go poller.Poll(ctx, wg)

	processor := db.NewRevertableAggregateProcessor(config.Processor, DB, processors...)
	wg.Add(1)
	go process.Process(ctx, wg, poller.DataCh(), processor)

	return nil
}
