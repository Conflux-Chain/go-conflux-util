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

	Health health.TimedCounterConfig
}

// RetriableProcessor operates on database till succeeded.
type RetriableProcessor struct {
	option Option
	db     *gorm.DB
	health *health.TimedCounter
}

func NewRetriableProcessor(db *gorm.DB, option Option) *RetriableProcessor {
	defaults.SetDefaults(&option)

	return &RetriableProcessor{
		option: option,
		db:     db,
		health: health.NewTimedCounter(option.Health),
	}
}

// Write executes the given op in a transaction. If failed, it will try again till succeeded.
func (processor *RetriableProcessor) Write(ctx context.Context, op Operation) {
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
