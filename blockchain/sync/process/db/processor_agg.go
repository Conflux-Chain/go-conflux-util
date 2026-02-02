package db

import (
	"context"

	"gorm.io/gorm"
)

// Processor is implemented by types that process data to update database.
type Processor[T any] interface {
	Process(data T) Operation
}

// AggregateProcessor aggregates multiple processor to process blockchain data in batch.
type AggregateProcessor[T any] struct {
	*RetriableProcessor

	processors []Processor[T]
}

func NewAggregateProcessor[T any](option Option, db *gorm.DB, processors ...Processor[T]) *AggregateProcessor[T] {
	return &AggregateProcessor[T]{
		RetriableProcessor: NewRetriableProcessor(db, option),
		processors:         processors,
	}
}

// Process implements the process.Processor[T] interface.
func (processor *AggregateProcessor[T]) Process(ctx context.Context, data T) {
	var ops []Operation

	for _, v := range processor.processors {
		op := v.Process(data)
		ops = append(ops, op)
	}

	processor.Write(ctx, ComposeOperation(ops...))
}
