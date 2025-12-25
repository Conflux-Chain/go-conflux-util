package db

import (
	"context"
	"sync"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/ctxutil"
	"github.com/mcuadros/go-defaults"
	"gorm.io/gorm"
)

type BatchOption struct {
	Processor Option

	BatchRows       int           `default:"3000"`
	BatchTimeout    time.Duration `default:"3s"`
	BufferSize      int           `default:"32"`
	CreateBatchSize int           `default:"1000"`
}

// BatchProcessor aggregates multiple processor to process blockchain data in batch.
//
// Generally, it is used during catch up phase.
type BatchProcessor[T any] struct {
	*AggregateProcessor[T]

	option BatchOption

	batch         *batchOperation
	lastBatchTime time.Time
	batchCh       chan *batchOperation

	mu sync.Mutex
}

func NewBatchProcessor[T any](option BatchOption, db *gorm.DB, processors ...Processor[T]) *BatchProcessor[T] {
	defaults.SetDefaults(option)

	return &BatchProcessor[T]{
		AggregateProcessor: NewAggregateProcessor(option.Processor, db, processors...),
		option:             option,
		batch:              newBatchOperation(option.CreateBatchSize),
		lastBatchTime:      time.Now(),
		batchCh:            make(chan *batchOperation, option.BufferSize),
	}
}

// Process implements the process.Processor[T] interface.
func (processor *BatchProcessor[T]) Process(ctx context.Context, data T) {
	for _, v := range processor.AggregateProcessor.processors {
		op := v.Process(data)
		processor.batch.Add(op)
	}

	processor.tryWriteOnce(ctx)
}

func (processor *BatchProcessor[T]) tryWriteOnce(ctx context.Context) {
	processor.mu.Lock()
	defer processor.mu.Unlock()

	// check rows and elapsed
	if rows := processor.batch.Rows(); rows < processor.option.BatchRows &&
		time.Since(processor.lastBatchTime) < processor.option.BatchTimeout {
		return
	}

	if err := ctxutil.WriteChannel(ctx, processor.batchCh, processor.batch); err != nil {
		return
	}

	// reset batch
	processor.batch = newBatchOperation(processor.option.CreateBatchSize)
	processor.lastBatchTime = time.Now()
}

// Write writes batch operations in a transaction. Generally, this is executed in a
// separate goroutine, and will terminate when the Poll goroutine completed.
func (processor *BatchProcessor[T]) Write(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	// ticker to check batch timeout
	ticker := time.NewTicker(processor.option.BatchTimeout / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			processor.tryWriteOnce(ctx)
		case batch, ok := <-processor.batchCh:
			if ok {
				processor.write(ctx, batch)
			} else {
				// write the cached batch if channel closed
				if processor.batch.Rows() > 0 {
					processor.write(ctx, processor.batch)
				}

				return
			}
		}
	}
}

// Close implements the process.Processor[T] interface.
//
// It will close the underlying channel so as to terminate the Write goroutine.
func (processor *BatchProcessor[T]) Close() {
	close(processor.batchCh)
}
