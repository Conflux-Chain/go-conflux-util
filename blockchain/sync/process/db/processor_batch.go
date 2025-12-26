package db

import (
	"context"
	"time"

	"github.com/mcuadros/go-defaults"
	"gorm.io/gorm"
)

// BatchProcessor is implemented by types that process data in batch.
//
// Note, thread-safe is not required in the implementations, since batch
// related methods are executed in a single thread.
type BatchProcessor[T any] interface {
	Processor[T]

	// BatchProcess processes the given data and returns the number of SQLs to be executed in batch.
	BatchProcess(data T) int

	// BatchExec executes SQLs in batch.
	BatchExec(tx *gorm.DB, createBatchSize int) error

	// BatchReset reset data for the next batch.
	BatchReset()
}

type BatchOption struct {
	Processor Option

	BatchSize       int           `default:"3000"`
	BatchTimeout    time.Duration `default:"3s"`
	CreateBatchSize int           `default:"1000"`
}

// BatchProcessor aggregates multiple processor to process blockchain data in batch.
//
// Generally, it is used during catch up phase.
type BatchAggregateProcessor[T any] struct {
	*AggregateProcessor[T]

	option        BatchOption
	processors    []BatchProcessor[T]
	lastBatchTime time.Time
}

func NewBatchAggregateProcessor[T any](option BatchOption, db *gorm.DB, processors ...BatchProcessor[T]) *BatchAggregateProcessor[T] {
	defaults.SetDefaults(option)

	innerProcessors := make([]Processor[T], 0, len(processors))
	for _, v := range processors {
		innerProcessors = append(innerProcessors, v)
	}

	return &BatchAggregateProcessor[T]{
		AggregateProcessor: NewAggregateProcessor(option.Processor, db, innerProcessors...),
		option:             option,
		lastBatchTime:      time.Now(),
	}
}

// Process implements the process.Processor[T] interface.
func (processor *BatchAggregateProcessor[T]) Process(ctx context.Context, data T) {
	var size int

	for _, v := range processor.processors {
		size += v.BatchProcess(data)
	}

	// Write database only if batch size reached or batch timeout.
	//
	// Note, if no more data polled, it will not write database even though batch timeout.
	// This situation will rarely happen in catch up phase, and the worst case is that
	// only one batch data not written into database in time.
	if size < processor.option.BatchSize && time.Since(processor.lastBatchTime) < processor.option.BatchTimeout {
		return
	}

	processor.blockingWrite(ctx, processor)

	// reset
	processor.lastBatchTime = time.Now()

	for _, v := range processor.processors {
		v.BatchReset()
	}
}

// Exec implements the Operation interface.
func (processor *BatchAggregateProcessor[T]) Exec(tx *gorm.DB) error {
	for _, v := range processor.processors {
		if err := v.BatchExec(tx, processor.option.CreateBatchSize); err != nil {
			return err
		}
	}

	return nil
}
