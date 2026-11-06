package process

import (
	"context"
	"sync"

	"github.com/Conflux-Chain/go-conflux-util/ctxutil"
)

// Processor defines how to process the polled blockchain data.
type Processor[T any] interface {
	// Process returns true when the processing step completes, or false when
	// cancellation interrupts it. Failures must be retried until completion or
	// cancellation; returning false must not be used to skip failed data.
	// For batch processors, completion may mean buffering data for a later commit.
	Process(ctx context.Context, data T) bool
}

type CatchUpProcessor[T any] interface {
	Processor[T]

	// OnCatchedUp is executed after the latest finalized block processed.
	OnCatchedUp(ctx context.Context)
}

// Process retrieves data from the given channel and processes data with given processor.
//
// Generally, it will be executed in a separate goroutine, and terminate if given context done or channel closed.
func Process[T any](ctx context.Context, wg *sync.WaitGroup, dataCh <-chan T, processor Processor[T]) {
	defer wg.Done()

	process(ctx, dataCh, processor)
}

func process[T any](ctx context.Context, dataCh <-chan T, processor Processor[T]) bool {
	for {
		select {
		case <-ctx.Done():
			return false
		case data, ok := <-dataCh:
			if ctxutil.IsDone(ctx) {
				return false
			}

			if !ok {
				return true
			}

			if !processor.Process(ctx, data) {
				return false
			}
		}
	}
}

// ProcessCatchUp processes the polled blockchain data from given data channel till the latest finalized block processed.
func ProcessCatchUp[T any](ctx context.Context, wg *sync.WaitGroup, dataCh <-chan T, processor CatchUpProcessor[T]) {
	defer wg.Done()

	if process(ctx, dataCh, processor) {
		processor.OnCatchedUp(ctx)
	}
}
