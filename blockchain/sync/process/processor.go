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

	// Close is executed when Process goroutine terminated.
	Close(ctx context.Context)
}

// Process retrieves data from the given channel and processes data with given processor.
//
// Generally, it will be executed in a separate goroutine, and terminate if given context done or channel closed.
func Process[T any](ctx context.Context, wg *sync.WaitGroup, dataCh <-chan T, processor Processor[T]) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case data, ok := <-dataCh:
			if !ok {
				processor.Close(ctx)
				return
			}

			processor.Process(ctx, data)

			// Check if context is done during processing, otherwise the for loop may continue to
			// process the next data when data channel has more data to process.
			if ctxutil.IsDone(ctx) {
				return
			}
		}
	}
}
