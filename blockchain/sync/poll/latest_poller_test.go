package poll

import (
	"context"
	"errors"
	"testing"

	"github.com/Conflux-Chain/go-conflux-util/blockchain/sync/poll/testutil"
	"github.com/stretchr/testify/require"
)

type observedLatestAdapter struct {
	*testutil.Adapter
	finalizedErr   error
	finalizedCalls int
	calls          []string
}

func (a *observedLatestAdapter) GetFinalizedBlock(ctx context.Context) (uint64, string, error) {
	a.calls = append(a.calls, "finalized")
	a.finalizedCalls++
	if a.finalizedErr != nil {
		return 0, "", a.finalizedErr
	}
	return a.Adapter.GetFinalizedBlock(ctx)
}

func (a *observedLatestAdapter) GetBlockData(ctx context.Context, number uint64) (testutil.Data, error) {
	a.calls = append(a.calls, "data")
	return a.Adapter.GetBlockData(ctx, number)
}

func TestLatestPollerFinalizedReplay(t *testing.T) {
	adapter := &observedLatestAdapter{Adapter: testutil.MustNewAdapter([]uint64{10}, []testutil.Data{
		{Number: 11, Hash: "DataHash-11", ParentHash: "DataHash-10"},
	})}
	poller, err := NewLatestPoller[testutil.Data](adapter, ReorgWindowParams{})
	require.NoError(t, err)
	ctx := context.Background()

	for number := uint64(0); number <= 10; number++ {
		data, ok, reorg, err := poller.pollOnce(ctx, poller.window.NextBlockNumber())
		require.NoError(t, err)
		require.True(t, ok)
		require.False(t, reorg)
		require.Equal(t, number, data.Number)
		require.Len(t, poller.window.blockNumber2Hashes, 1)
		require.Equal(t, number, poller.window.finalized)
	}
	require.Equal(t, 1, adapter.finalizedCalls)
	// Genesis needs no finality query; block 1 is fetched after observing finality.
	require.Equal(t, []string{"data", "finalized", "data"}, adapter.calls[:3])

	// Beyond the cached boundary, refresh finality and retain unfinalized history.
	data, ok, reorg, err := poller.pollOnce(ctx, poller.window.NextBlockNumber())
	require.NoError(t, err)
	require.True(t, ok)
	require.False(t, reorg)
	require.Equal(t, uint64(11), data.Number)
	require.Equal(t, 2, adapter.finalizedCalls)
	require.Equal(t, []string{"finalized", "data"}, adapter.calls[len(adapter.calls)-2:])
	require.Len(t, poller.window.blockNumber2Hashes, 2)
	require.Equal(t, uint64(10), poller.window.finalized)
}

func TestLatestPollerRestoredForkBelowFinality(t *testing.T) {
	adapter := &observedLatestAdapter{Adapter: testutil.MustNewAdapter([]uint64{10}, nil)}
	poller, err := NewLatestPoller[testutil.Data](adapter, ReorgWindowParams{
		FinalizedBlockNumber: 5,
		FinalizedBlockHash:   "DataHash-5",
		LatestBlocks:         map[uint64]string{6: "old-6", 7: "old-7"},
	})
	require.NoError(t, err)
	ctx := context.Background()

	// Finality does not authorize discarding a restored fork before reconnecting.
	for next := uint64(7); next >= 6; next-- {
		_, ok, reorg, err := poller.pollOnce(ctx, poller.window.NextBlockNumber())
		require.NoError(t, err)
		require.False(t, ok)
		require.True(t, reorg)
		require.Equal(t, next, poller.window.NextBlockNumber())
		require.Equal(t, uint64(5), poller.window.finalized)
	}

	data, ok, reorg, err := poller.pollOnce(ctx, poller.window.NextBlockNumber())
	require.NoError(t, err)
	require.True(t, ok)
	require.False(t, reorg)
	require.Equal(t, uint64(6), data.Number)
	require.Equal(t, map[uint64]string{6: "DataHash-6"}, poller.window.blockNumber2Hashes)
	require.Equal(t, 1, adapter.finalizedCalls)
}

func TestLatestPollerFinalityQueryFailure(t *testing.T) {
	adapter := &observedLatestAdapter{
		Adapter:      testutil.MustNewAdapter([]uint64{10}, nil),
		finalizedErr: errors.New("finality unavailable"),
	}
	poller, err := NewLatestPoller[testutil.Data](adapter, ReorgWindowParams{})
	require.NoError(t, err)
	ctx := context.Background()

	for number := uint64(0); number <= 1; number++ {
		data, ok, reorg, err := poller.pollOnce(ctx, poller.window.NextBlockNumber())
		require.NoError(t, err)
		require.True(t, ok)
		require.False(t, reorg)
		require.Equal(t, number, data.Number)
	}
	require.Len(t, poller.window.blockNumber2Hashes, 2)

	// A later successful query permits compaction without skipping accepted blocks.
	adapter.finalizedErr = nil
	data, ok, reorg, err := poller.pollOnce(ctx, poller.window.NextBlockNumber())
	require.NoError(t, err)
	require.True(t, ok)
	require.False(t, reorg)
	require.Equal(t, uint64(2), data.Number)
	require.Equal(t, map[uint64]string{2: "DataHash-2"}, poller.window.blockNumber2Hashes)
}
