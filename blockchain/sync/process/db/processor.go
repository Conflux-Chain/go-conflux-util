package db

import (
	"context"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/ctxutil"
	"github.com/Conflux-Chain/go-conflux-util/health"
	"github.com/Conflux-Chain/go-conflux-util/log"
	"github.com/mcuadros/go-defaults"
	"gorm.io/gorm"
)

var ModuleName = "sync.process.db"

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

// Write executes op in a transaction, retrying failures until it succeeds or ctx is canceled.
// It returns true if the transaction committed, or false if cancellation stopped the retries.
func (processor *RetriableProcessor) Write(ctx context.Context, op Operation) bool {
	for {
		if ctxutil.IsDone(ctx) {
			return false
		}

		err := processor.db.Transaction(func(tx *gorm.DB) error {
			return op.Exec(tx)
		})

		processor.health.LogOnError(err, "Process blockchain data in Database")

		if err == nil {
			return true
		}

		log.WithModule(ModuleName).WithError(err).Debug("Failed to write database")

		if err = ctxutil.Sleep(ctx, processor.option.RetryInterval); err != nil {
			return false
		}
	}
}
