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

type operationFunc func(*gorm.DB) error

func (f operationFunc) Exec(tx *gorm.DB) error { return f(tx) }

func TestWriteRetriesUntilCommit(t *testing.T) {
	database := newCreateTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	attempts := 0
	op := operationFunc(func(tx *gorm.DB) error {
		attempts++
		if err := tx.Create(&createRecord{ID: 1, Name: "Alice"}).Error; err != nil {
			return err
		}
		if attempts == 1 {
			return errors.New("temporary failure")
		}
		return nil
	})
	processor := dbproc.NewRetriableProcessor(database, dbproc.Option{RetryInterval: time.Millisecond})
	assert.True(t, processor.Write(ctx, op))
	assert.Equal(t, 2, attempts)
	var count int64
	require.NoError(t, database.Model(&createRecord{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestWriteCancellationResult(t *testing.T) {
	for _, tc := range []struct {
		name          string
		cancelBefore  bool
		failOperation bool
		wantCommitted bool
	}{
		{name: "already canceled", cancelBefore: true},
		{name: "canceled after failure", failOperation: true},
		{name: "committed during cancellation", wantCommitted: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			database := newCreateTestDB(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.cancelBefore {
				cancel()
			}
			attempts := 0
			op := operationFunc(func(tx *gorm.DB) error {
				attempts++
				if err := tx.Create(&createRecord{ID: 1}).Error; err != nil {
					return err
				}
				cancel()
				if tc.failOperation {
					return errors.New("write failed")
				}
				return nil
			})
			processor := dbproc.NewRetriableProcessor(database, dbproc.Option{})
			assert.Equal(t, tc.wantCommitted, processor.Write(ctx, op))
			if tc.cancelBefore {
				assert.Zero(t, attempts)
			} else {
				assert.Equal(t, 1, attempts)
			}
			var count int64
			require.NoError(t, database.Model(&createRecord{}).Count(&count).Error)
			if tc.wantCommitted {
				assert.Equal(t, int64(1), count)
			} else {
				assert.Zero(t, count)
			}
		})
	}
}
