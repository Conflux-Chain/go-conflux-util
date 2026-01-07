package sync

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/blockchain/sync/poll"
	"github.com/Conflux-Chain/go-conflux-util/blockchain/sync/poll/testutil"
	"github.com/Conflux-Chain/go-conflux-util/blockchain/sync/process/db"
	"github.com/Conflux-Chain/go-conflux-util/parallel"
	"github.com/Conflux-Chain/go-conflux-util/store"
	"github.com/stretchr/testify/assert"
)

func TestCatchUpDB(t *testing.T) {
	adapter := testutil.MustNewAdapter([]uint64{3, 5}, nil)

	storeConfig := store.NewMemoryConfig()
	DB := storeConfig.MustOpenOrCreate()

	processor := newTestDBProcessor()

	nextBlockNumber := CatchUpDB(
		context.Background(),
		CatchupParamsDB[testutil.Data]{
			Adapter: adapter,
			Poller: poll.CatchUpOption{
				Parallel: poll.ParallelOption{
					SerialOption: parallel.SerialOption{
						Window: 5,
					},
				},
			},
			Processor: db.BatchOption{
				BatchSize: 3, // 1 batch contains 3 blocks
			},
			DB:              DB,
			NextBlockNumber: 2, // start to sync from block 2
		},
		processor,
	)

	assert.Equal(t, uint64(6), nextBlockNumber)

	processor.assertData(t,
		[][]uint64{
			{2, 3, 4},
			{5},
		},
		nil,
		nil,
	)
}

func TestSyncFinalizedDB(t *testing.T) {
	adapter := testutil.MustNewAdapter([]uint64{3, 5}, nil)

	storeConfig := store.NewMemoryConfig()
	DB := storeConfig.MustOpenOrCreate()

	processor := newTestDBProcessor()

	var wg sync.WaitGroup

	StartFinalizedDB(context.Background(), &wg, ParamsDB[testutil.Data]{
		Adapter:         adapter,
		DB:              DB,
		NextBlockNumber: 2, // start to sync from block 2
	}, processor)

	// wait for poll-and-process
	time.Sleep(500 * time.Millisecond)

	processor.assertData(t,
		nil,
		[]uint64{2, 3, 4, 5},
		nil,
	)
}

func TestSyncLatestDB(t *testing.T) {
	// finalized block number: 5
	// latest block number: 9, with 2 reorgs
	adapter := testutil.MustNewAdapter([]uint64{3, 5}, []testutil.Data{
		{Number: 6, Hash: "DataHash-6", ParentHash: "DataHash-5"},
		{Number: 7, Hash: "DataHash-7", ParentHash: "DataHash-6"},

		// revert block 7
		{Number: 8, Hash: "DataHash-88", ParentHash: "DataHash-77"},
		{Number: 7, Hash: "DataHash-77", ParentHash: "DataHash-6"},
		{Number: 8, Hash: "DataHash-88", ParentHash: "DataHash-77"},

		// revert blocks 7 & 8
		{Number: 9, Hash: "DataHash-999", ParentHash: "DataHash-888"},
		{Number: 8, Hash: "DataHash-888", ParentHash: "DataHash-777"},
		{Number: 7, Hash: "DataHash-777", ParentHash: "DataHash-6"},
		{Number: 8, Hash: "DataHash-888", ParentHash: "DataHash-777"},
		{Number: 9, Hash: "DataHash-999", ParentHash: "DataHash-888"},
	})

	storeConfig := store.NewMemoryConfig()
	DB := storeConfig.MustOpenOrCreate()

	processor := newTestDBProcessor()

	var wg sync.WaitGroup

	StartLatestDB(context.Background(), &wg, ParamsDB[testutil.Data]{
		Adapter:         adapter,
		DB:              DB,
		NextBlockNumber: 2, // start to sync from block 2
	}, processor)

	// wait for poll-and-process
	time.Sleep(500 * time.Millisecond)

	processor.assertData(t,
		nil,
		[]uint64{2, 3, 4, 5, 6, 7, 8, 9},
		// 2 reorgs
		[][]uint64{
			{7},
			{7, 8},
		},
	)
}
