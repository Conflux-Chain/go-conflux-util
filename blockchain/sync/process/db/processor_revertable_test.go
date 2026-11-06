package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Conflux-Chain/go-conflux-util/blockchain/sync/poll"
	dbproc "github.com/Conflux-Chain/go-conflux-util/blockchain/sync/process/db"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

type canceledRevertProcessor struct {
	cancel        context.CancelFunc
	processCalled bool
}

func (p *canceledRevertProcessor) Revert(int) dbproc.Operation {
	return operationFunc(func(*gorm.DB) error {
		p.cancel()
		return errors.New("revert failed")
	})
}

func (p *canceledRevertProcessor) Process(int) dbproc.Operation {
	p.processCalled = true
	return operationFunc(func(*gorm.DB) error { return nil })
}

func TestCanceledRevertStopsReplacement(t *testing.T) {
	database := newCreateTestDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	custom := &canceledRevertProcessor{cancel: cancel}
	processor := dbproc.NewRevertableAggregateProcessor[int](dbproc.Option{}, database, custom)
	assert.False(t, processor.Process(ctx, poll.Revertable[int]{Data: 1, Reverted: true}))
	assert.False(t, custom.processCalled)
}
