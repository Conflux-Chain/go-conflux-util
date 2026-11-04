package db_test

import (
	"context"
	"errors"
	"testing"
	"time"

	dbproc "github.com/Conflux-Chain/go-conflux-util/blockchain/sync/process/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type testBatchProcessor struct {
	op      operationFunc
	pending int
	reset   bool
}

func (p *testBatchProcessor) BatchProcess(int) int {
	p.pending++
	return p.pending
}

func (p *testBatchProcessor) BatchExec(tx *gorm.DB, _ int) error { return p.op(tx) }

func (p *testBatchProcessor) BatchReset() {
	p.pending = 0
	p.reset = true
}

func TestBatchResetOnlyAfterCommit(t *testing.T) {
	for _, cancelWrite := range []bool{false, true} {
		name := "committed"
		if cancelWrite {
			name = "canceled"
		}
		t.Run(name, func(t *testing.T) {
			database := newPersistentTestDB(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			batch := &testBatchProcessor{op: func(tx *gorm.DB) error {
				if err := tx.Create(&createRecord{ID: 1, Name: "Alice"}).Error; err != nil {
					return err
				}
				if cancelWrite {
					cancel()
					return errors.New("write failed")
				}
				return nil
			}}
			processor := dbproc.NewBatchAggregateProcessor[int](dbproc.BatchOption{BatchSize: 1}, database, batch)
			assert.Equal(t, !cancelWrite, processor.Process(ctx, 1))
			var count int64
			require.NoError(t, database.Model(&createRecord{}).Count(&count).Error)
			if cancelWrite {
				assert.False(t, batch.reset)
				assert.Equal(t, 1, batch.pending)
				assert.Zero(t, count)
			} else {
				assert.True(t, batch.reset)
				assert.Zero(t, batch.pending)
				assert.Equal(t, int64(1), count)
			}
		})
	}
}

func TestBatchProcessBuffersBeforeCommit(t *testing.T) {
	database := newCreateTestDB(t)
	batch := &testBatchProcessor{op: func(tx *gorm.DB) error {
		return tx.Create(&createRecord{ID: 1}).Error
	}}
	processor := dbproc.NewBatchAggregateProcessor[int](dbproc.BatchOption{
		BatchSize: 10, BatchTimeout: time.Hour,
	}, database, batch)
	assert.True(t, processor.Process(context.Background(), 1))
	assert.Equal(t, 1, batch.pending)
	assert.False(t, batch.reset)
	var count int64
	require.NoError(t, database.Model(&createRecord{}).Count(&count).Error)
	assert.Zero(t, count)

	processor.OnCatchedUp(context.Background())
	require.NoError(t, database.Model(&createRecord{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
	assert.True(t, batch.reset)
}

func TestCanceledBatchDoesNotWrite(t *testing.T) {
	database := newCreateTestDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	batch := &testBatchProcessor{op: func(*gorm.DB) error {
		t.Error("unexpected write")
		return nil
	}}
	processor := dbproc.NewBatchAggregateProcessor[int](dbproc.BatchOption{BatchSize: 1}, database, batch)
	assert.False(t, processor.Process(ctx, 1))
	assert.Equal(t, 1, batch.pending)
	assert.False(t, batch.reset)
}
