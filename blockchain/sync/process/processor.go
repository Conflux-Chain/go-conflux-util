package process

import (
	"context"
	"sync"

	"github.com/Conflux-Chain/go-conflux-util/ctxutil"
)

// Processor defines how to process the polled blockchain data.
type Processor[T any] interface {
	// Process processes the given data.
	Process(ctx context.Context, data T)
}

type CatchUpProcessor[T any] interface {
	Processor[T]

	// OnCatchedUp is executed after the latest finalized block processed.
	OnCatchedUp(ctx context.Context)
}

// Process retrieves data from the given channel and processes data with given processor.
//
// Generally, it will be executed in a separate goroutine, and terminate if given context done or channel closed.
//
// It returns true if the given channel closed. Otherwise false, if the given context done.
func Process[T any](ctx context.Context, wg *sync.WaitGroup, dataCh <-chan T, processor Processor[T]) bool {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return false
		case data, ok := <-dataCh:
			if !ok {
				return true
			}

			processor.Process(ctx, data)

			// Check if context is done during processing, otherwise the for loop may continue to
			// process the next data when data channel has more data to process.
			if ctxutil.IsDone(ctx) {
				return false
			}
		}
	}
}

// ProcessCatchUp processes the polled blockchain data from given data channel till the latest finalized block processed.
func ProcessCatchUp[T any](ctx context.Context, wg *sync.WaitGroup, dataCh <-chan T, processor CatchUpProcessor[T]) {
	if Process(ctx, wg, dataCh, processor) {
		processor.OnCatchedUp(ctx)
	}
}
