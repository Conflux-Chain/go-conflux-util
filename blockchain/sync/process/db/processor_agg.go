package db

import (
	"context"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/ctxutil"
	"github.com/Conflux-Chain/go-conflux-util/health"
	"github.com/mcuadros/go-defaults"
	"gorm.io/gorm"
)

type Option struct {
	RetryInterval time.Duration `default:"3s"`

	Health health.CounterConfig
}

// AggregateProcessor aggregates multiple processor to process blockchain data in batch.
type AggregateProcessor[T any] struct {
	option     Option
	db         *gorm.DB
	processors []Processor[T]
	health     *health.Counter
}

func NewAggregateProcessor[T any](option Option, db *gorm.DB, processors ...Processor[T]) *AggregateProcessor[T] {
	defaults.SetDefaults(&option)

	return &AggregateProcessor[T]{
		option:     option,
		db:         db,
		processors: processors,
		health:     health.NewCounter(option.Health),
	}
}

// Process implements the process.Processor[T] interface.
func (processor *AggregateProcessor[T]) Process(ctx context.Context, data T) {
	var ops []Operation

	for _, v := range processor.processors {
		op := v.Process(data)
		ops = append(ops, op)
	}

	processor.write(ctx, ComposeOperation(ops...))
}

// write executes the give op in a transaction.
func (processor *AggregateProcessor[T]) write(ctx context.Context, op Operation) {
	for {
		err := processor.db.Transaction(func(tx *gorm.DB) error {
			return op.Exec(tx)
		})

		processor.health.LogOnError(err, "Process blockchain data in Database")

		if err == nil {
			return
		}

		if err = ctxutil.Sleep(ctx, processor.option.RetryInterval); err != nil {
			return
		}
	}
}

// Close implements the process.Processor[T] interface.
func (processor *AggregateProcessor[T]) Close() {
	// do nothing
}
