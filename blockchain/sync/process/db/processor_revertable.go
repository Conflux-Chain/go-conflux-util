package db

import (
	"context"

	"github.com/Conflux-Chain/go-conflux-util/blockchain/sync/poll"
	"gorm.io/gorm"
)

// RevertableProcessor is implemented by types that process revertable data.
type RevertableProcessor[T any] interface {
	Processor[T]

	// Revert deletes data from database of given data block number.
	Revert(data T) Operation
}

// RevertableAggregateProcessor aggregates multiple processor to process blockchain data in batch,
// and supports to process the reverted data when chain reorg happened.
type RevertableAggregateProcessor[T any] struct {
	*AggregateProcessor[T]

	processors []RevertableProcessor[T]
}

func NewRevertableAggregateProcessor[T any](option Option, db *gorm.DB, processors ...RevertableProcessor[T]) *RevertableAggregateProcessor[T] {
	innerProcessors := make([]Processor[T], 0, len(processors))
	for _, v := range processors {
		innerProcessors = append(innerProcessors, v)
	}

	return &RevertableAggregateProcessor[T]{
		AggregateProcessor: NewAggregateProcessor(option, db, innerProcessors...),

		processors: processors,
	}
}

// Process implements the process.Processor[poll.Revertable[T]] interface.
func (processor *RevertableAggregateProcessor[T]) Process(ctx context.Context, data poll.Revertable[T]) {
	if data.Reverted {
		var ops []Operation

		for _, v := range processor.processors {
			op := v.Revert(data.Data)
			ops = append(ops, op)
		}

		processor.write(ctx, ComposeOperation(ops...))
	}

	processor.AggregateProcessor.Process(ctx, data.Data)
}
