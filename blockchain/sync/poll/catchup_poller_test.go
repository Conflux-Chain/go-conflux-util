package poll

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/parallel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type catchUpData uint64

func (catchUpData) Size() int { return 1 }

type gatedCatchUpAdapter struct {
	blocks  int
	gate    <-chan struct{}
	started chan uint64
}

func (a *gatedCatchUpAdapter) GetFinalizedBlock(context.Context) (uint64, string, error) {
	return uint64(a.blocks - 1), fmt.Sprint(a.blocks - 1), nil
}

func (a *gatedCatchUpAdapter) GetLatestBlockNumber(ctx context.Context) (uint64, error) {
	number, _, err := a.GetFinalizedBlock(ctx)
	return number, err
}

func (a *gatedCatchUpAdapter) GetBlockData(ctx context.Context, n uint64) (catchUpData, error) {
	a.started <- n
	if n == 0 {
		select {
		case <-a.gate:
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
	return catchUpData(n), nil
}

func (*gatedCatchUpAdapter) GetBlockHash(data catchUpData) string {
	return fmt.Sprint(data)
}

func (*gatedCatchUpAdapter) GetParentBlockHash(data catchUpData) string {
	return fmt.Sprint(data - 1)
}

func TestCatchUpPrefetchWindow(t *testing.T) {
	previousProcs := runtime.GOMAXPROCS(2)
	t.Cleanup(func() { runtime.GOMAXPROCS(previousProcs) })

	for _, tc := range []struct {
		name     string
		routines int
		window   int
		blocks   int
		want     int
	}{
		{"default window", 4, 0, 32, 8},
		{"default concurrency", 0, 0, 32, 4},
		{"explicit window", 2, 6, 32, 6},
		{"window below concurrency", 4, 2, 32, 4},
		{"negative window remains unlimited", 4, -1, 32, 32},
		{"single remaining block", 4, 0, 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			gate := make(chan struct{})
			adapter := &gatedCatchUpAdapter{
				blocks: tc.blocks, gate: gate, started: make(chan uint64, tc.blocks),
			}
			option := CatchUpOption{Parallel: ParallelOption{SerialOption: parallel.SerialOption{
				Routines: tc.routines, Window: tc.window,
			}}}
			option.Buffer.Capacity = 1
			option.Buffer.MaxBytes = 1
			poller := NewCatchUpPoller[catchUpData](adapter, 0, option)
			var wg sync.WaitGroup
			wg.Add(1)
			go poller.Poll(ctx, &wg)
			var received []catchUpData
			drained := make(chan struct{})
			go func() {
				for data := range poller.DataCh() {
					received = append(received, data)
				}
				close(drained)
			}()
			t.Cleanup(func() {
				cancel()
				wg.Wait()
				<-drained
			})

			for i := 0; i < tc.want; i++ {
				select {
				case <-adapter.started:
				case <-ctx.Done():
					t.Fatal("expected prefetch window was not filled")
				}
			}
			select {
			case n := <-adapter.started:
				t.Errorf("block %d started beyond the window while block 0 was blocked", n)
			case <-time.After(100 * time.Millisecond):
			}

			close(gate)
			wg.Wait()
			<-drained
			require.NoError(t, ctx.Err())
			want := make([]catchUpData, tc.blocks)
			for i := range want {
				want[i] = catchUpData(i)
			}
			assert.Equal(t, want, received)
			assert.Equal(t, uint64(tc.blocks), poller.NextBlockNumber())
		})
	}
}
