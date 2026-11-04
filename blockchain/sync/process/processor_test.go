package process_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/blockchain/sync/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type catchUpProcessor struct {
	process  func(context.Context, int) bool
	caughtUp bool
}

func (p *catchUpProcessor) Process(ctx context.Context, data int) bool {
	return p.process(ctx, data)
}

func (p *catchUpProcessor) OnCatchedUp(context.Context) { p.caughtUp = true }

func runCatchUp(ctx context.Context, dataCh <-chan int, p *catchUpProcessor) {
	var wg sync.WaitGroup
	wg.Add(1)
	process.ProcessCatchUp(ctx, &wg, dataCh, p)
	wg.Wait()
}

func TestCatchUpCompletesInOrder(t *testing.T) {
	dataCh := make(chan int, 3)
	for _, n := range []int{10, 11, 12} {
		dataCh <- n
	}
	close(dataCh)
	var processed []int
	p := &catchUpProcessor{process: func(_ context.Context, n int) bool {
		processed = append(processed, n)
		return true
	}}
	runCatchUp(context.Background(), dataCh, p)
	assert.Equal(t, []int{10, 11, 12}, processed)
	assert.True(t, p.caughtUp)
}

func TestCatchUpCancellationTakesPriority(t *testing.T) {
	for _, buffered := range []bool{false, true} {
		// Exercise both ready select cases: cancellation and data/channel closure.
		for i := 0; i < 64; i++ {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			dataCh := make(chan int, 1)
			if buffered {
				dataCh <- 10
			}
			close(dataCh)
			p := &catchUpProcessor{process: func(context.Context, int) bool {
				t.Error("processing started after cancellation")
				return true
			}}
			runCatchUp(ctx, dataCh, p)
			require.False(t, p.caughtUp, "cancellation must not trigger normal completion")
		}
	}
}

func TestCatchUpStopsAfterProcessing(t *testing.T) {
	for _, tc := range []struct {
		name      string
		completed bool
		cancel    bool
	}{
		{name: "false result"},
		{name: "canceled during processing", cancel: true},
		{name: "completed during cancellation", completed: true, cancel: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			dataCh := make(chan int, 2)
			dataCh <- 10
			dataCh <- 11
			close(dataCh)
			var processed []int
			p := &catchUpProcessor{process: func(_ context.Context, n int) bool {
				processed = append(processed, n)
				if tc.cancel {
					cancel()
				}
				return tc.completed
			}}
			runCatchUp(ctx, dataCh, p)
			assert.Equal(t, []int{10}, processed)
			assert.False(t, p.caughtUp)
		})
	}
}

func TestCatchUpCancellationWhileWaiting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dataCh := make(chan int)
	p := &catchUpProcessor{process: func(context.Context, int) bool {
		t.Error("unexpected data")
		return true
	}}
	done := make(chan struct{})
	go func() {
		runCatchUp(ctx, dataCh, p)
		close(done)
	}()
	cancel()
	select {
	case <-done:
		assert.False(t, p.caughtUp)
	case <-time.After(time.Second):
		close(dataCh)
		<-done
		t.Fatal("consumer did not exit while waiting for data")
	}
}
