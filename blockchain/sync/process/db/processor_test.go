package db_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	dbproc "github.com/Conflux-Chain/go-conflux-util/blockchain/sync/process/db"
	"github.com/Conflux-Chain/go-conflux-util/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type operationFunc func(*gorm.DB) error

func (f operationFunc) Exec(tx *gorm.DB) error { return f(tx) }

// Canceled transactions may discard their connection. A file database keeps
// committed data available for assertions even if that connection is replaced.
func newPersistentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	config := store.Config{
		Sqlite:       &store.SqliteConfig{Path: filepath.Join(t.TempDir(), "test.db")},
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		LogLevel:     "warn",
	}
	database, err := config.OpenOrCreate(&createRecord{})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, sqlDB.Close()) })
	return database
}

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
	}{
		{name: "already canceled", cancelBefore: true},
		{name: "canceled after failure", failOperation: true},
		{name: "canceled before commit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			database := newPersistentTestDB(t)
			existing := createRecord{ID: 99, Name: "committed"}
			require.NoError(t, database.Create(&existing).Error)
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
				assert.ErrorIs(t, tx.Statement.Context.Err(), context.Canceled)
				assert.Error(t, tx.Exec("SELECT 1").Error)
				if tc.failOperation {
					return errors.New("write failed")
				}
				return nil
			})
			processor := dbproc.NewRetriableProcessor(database, dbproc.Option{})
			assert.False(t, processor.Write(ctx, op))
			if tc.cancelBefore {
				assert.Zero(t, attempts)
			} else {
				assert.Equal(t, 1, attempts)
			}
			var rows []createRecord
			require.NoError(t, database.Order("id").Find(&rows).Error)
			assert.Equal(t, []createRecord{existing}, rows)
		})
	}
}
