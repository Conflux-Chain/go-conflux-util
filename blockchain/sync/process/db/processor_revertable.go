package db

import (
	"context"
	"slices"

	"github.com/Conflux-Chain/go-conflux-util/blockchain/sync/poll"
	"github.com/Conflux-Chain/go-conflux-util/log"
	"gorm.io/gorm"
)

// RevertableProcessor is implemented by types that process revertable data.
type RevertableProcessor[T any] interface {
	Processor[T]

	// Revert returns an operation that rolls back old-chain data at and above
	// the block number represented by data. The data belongs to the replacement block.
	// Any persisted sync progress and related state must be rolled back in the
	// same transaction so that synchronization can resume from the retained data.
	Revert(data T) Operation
}

// RevertableAggregateProcessor aggregates multiple processor to process blockchain data in batch,
// and supports to process the reverted data when chain reorg happened.
type RevertableAggregateProcessor[T any] struct {
	*AggregateProcessor[T]

	processors []RevertableProcessor[T]
}

func NewRevertableAggregateProcessor[T any](option Option, db *gorm.DB, processors ...RevertableProcessor[T]) *RevertableAggregateProcessor[T] {
	// push in order, and pop in reversed order
	pushProcessors := make([]Processor[T], 0, len(processors))
	popProcessors := make([]RevertableProcessor[T], 0, len(processors))

	for _, v := range processors {
		pushProcessors = append(pushProcessors, v)
		popProcessors = append(popProcessors, v)
	}

	slices.Reverse(popProcessors)

	return &RevertableAggregateProcessor[T]{
		AggregateProcessor: NewAggregateProcessor(option, db, pushProcessors...),

		processors: popProcessors,
	}
}

// Process implements the process.Processor[poll.Revertable[T]] interface.
//
// Reorg rollback and replacement insertion intentionally use separate transactions.
// A committed rollback is a valid restart point even if replacement insertion is
// interrupted. On restart, load committed database progress and recheck recent
// blocks for reorgs before continuing synchronization.
func (processor *RevertableAggregateProcessor[T]) Process(ctx context.Context, data poll.Revertable[T]) bool {
	if data.Reverted {
		var ops []Operation

		for _, v := range processor.processors {
			op := v.Revert(data.Data)
			ops = append(ops, op)
		}

		if !processor.Write(ctx, ComposeOperation(ops...)) {
			return false
		}

		log.WithModule(ModuleName).Debug("Succeeded to process reverted blockchain data")
	}

	// Reuse normal processing and retries without repeating the committed rollback.
	return processor.AggregateProcessor.Process(ctx, data.Data)
}
