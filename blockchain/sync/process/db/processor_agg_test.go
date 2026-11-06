package db_test

import (
	"context"
	"errors"
	"testing"

	dbproc "github.com/Conflux-Chain/go-conflux-util/blockchain/sync/process/db"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

type operationProcessor struct{ op dbproc.Operation }

func (p operationProcessor) Process(int) dbproc.Operation { return p.op }

func TestAggregateProcessResult(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		database := newCreateTestDB(t)
		ctx, cancel := context.WithCancel(context.Background())
		op := operationFunc(func(tx *gorm.DB) error {
			if canceled {
				cancel()
				return errors.New("write failed")
			}
			return tx.Create(&createRecord{ID: 1}).Error
		})
		processor := dbproc.NewAggregateProcessor[int](dbproc.Option{}, database, operationProcessor{op})
		assert.Equal(t, !canceled, processor.Process(ctx, 1))
		cancel()
	}
}
