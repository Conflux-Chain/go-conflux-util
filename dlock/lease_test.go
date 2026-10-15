package dlock

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestMySQLAcquisitionWaitBudget(t *testing.T) {
	for _, expired := range []bool{false, true} {
		t.Run(fmt.Sprintf("expired=%t", expired), func(t *testing.T) {
			backend := mysqlTestTable(t)
			key, nonce := testID(t), testID(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			var workers sync.WaitGroup
			defer func() { cancel(); workers.Wait() }()
			require.NoError(t, backend.db.Create(&Dlock{Key: key, Nonce: testID(t), ExpiredAt: time.Now().Add(-time.Hour)}).Error)
			tx := backend.db.WithContext(ctx).Begin()
			require.NoError(t, tx.Error)
			defer tx.Rollback()
			require.NoError(t, tx.Exec("SELECT * FROM ? WHERE `key` = ? FOR UPDATE", clause.Table{Name: backend.db.Statement.Table}, key).Error)
			lease := 5 * time.Second
			if expired {
				lease = 200 * time.Millisecond
			}
			require.NoError(t, backend.db.Connection(func(db *gorm.DB) error {
				db = db.Session(&gorm.Session{})
				var connectionID uint64
				require.NoError(t, db.Raw("SELECT CONNECTION_ID()").Scan(&connectionID).Error)
				type result struct {
					deadline time.Time
					err      error
				}
				results := make(chan result, 1)
				workers.Add(1)
				go func() {
					defer workers.Done()
					deadline, err := NewLockManager(NewMySQLBackend(db)).AcquireWithDeadline(ctx, NewLockIntent(key, nonce, lease))
					results <- result{deadline, err}
				}()
				require.Eventually(t, func() bool {
					var waiting int64
					err := backend.db.Raw(`SELECT COUNT(*) FROM performance_schema.data_lock_waits w
      JOIN performance_schema.threads t ON t.THREAD_ID = w.REQUESTING_THREAD_ID
      WHERE t.PROCESSLIST_ID = ?`, connectionID).Scan(&waiting).Error
					return err == nil && waiting > 0
				}, 3*time.Second, 10*time.Millisecond, "acquisition must actually wait for the row lock")
				observedWait := time.Now()
				if expired {
					<-time.After(lease + 100*time.Millisecond)
				} else {
					<-time.After(100 * time.Millisecond)
				}
				require.NoError(t, tx.Commit().Error)
				got := <-results
				if expired {
					assert.ErrorIs(t, got.err, ErrLockAcquisitionFailed)
					assert.True(t, got.deadline.IsZero())
					var count int64
					require.NoError(t, backend.db.Model(&Dlock{}).Where("`key` = ?", key).Count(&count).Error)
					assert.Zero(t, count, "discard the expired acquisition")
				} else {
					require.NoError(t, got.err)
					assert.True(t, got.deadline.After(time.Now()))
					assert.False(t, got.deadline.After(observedWait.Add(lease-time.Millisecond)))
					before := time.Now()
					var remaining int64
					require.NoError(t, backend.db.Raw("SELECT TIMESTAMPDIFF(MICROSECOND,UTC_TIMESTAMP(3),expired_at) FROM ? WHERE `key` = ?", clause.Table{Name: backend.db.Statement.Table}, key).Scan(&remaining).Error)
					assert.LessOrEqual(t, got.deadline.Sub(before), time.Duration(remaining)*time.Microsecond)
				}
				return nil
			}))
		})
	}
}

// Delay return after a real committed insert/update. A successor may already own
// the expired lease by the time the original caller checks its budget.
func TestMySQLDelayedWriteDoesNotReleaseSuccessor(t *testing.T) {
	for _, update := range []bool{false, true} {
		t.Run(fmt.Sprintf("update=%t", update), func(t *testing.T) {
			backend := mysqlTestTable(t)
			key, nonceA, nonceB := testID(t), testID(t), testID(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			var workers sync.WaitGroup
			defer func() { cancel(); workers.Wait() }()
			if update {
				require.NoError(t, backend.db.Create(&Dlock{Key: key, Nonce: testID(t), ExpiredAt: time.Now().Add(-time.Hour)}).Error)
			}
			written, release := make(chan struct{}), make(chan struct{})
			callback := "dlock_delay_" + testID(t)
			delay := func(tx *gorm.DB) {
				values, ok := tx.Statement.Dest.(map[string]interface{})
				if tx.Statement.Table != backend.db.Statement.Table || !ok || values["nonce"] != nonceA || tx.Error != nil || tx.RowsAffected == 0 {
					return
				}
				close(written)
				select {
				case <-release:
				case <-ctx.Done():
				}
			}
			if update {
				require.NoError(t, backend.db.Callback().Update().After("gorm:commit_or_rollback_transaction").Register(callback, delay))
				t.Cleanup(func() { assert.NoError(t, backend.db.Callback().Update().Remove(callback)) })
			} else {
				require.NoError(t, backend.db.Callback().Create().After("gorm:commit_or_rollback_transaction").Register(callback, delay))
				t.Cleanup(func() { assert.NoError(t, backend.db.Callback().Create().Remove(callback)) })
			}
			type result struct {
				ok  bool
				err error
			}
			results := make(chan result, 1)
			workers.Add(1)
			go func() {
				defer workers.Done()
				ok, err := backend.WriteEntry(ctx, key, nonceA, 200*time.Millisecond)
				results <- result{ok, err}
			}()
			select {
			case <-written:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			// The lease begins before the callback; this timer deliberately expires it.
			<-time.After(300 * time.Millisecond)
			ok, err := backend.WriteEntry(ctx, key, nonceB, time.Minute)
			require.NoError(t, err)
			require.True(t, ok)
			close(release)
			got := <-results
			assert.False(t, got.ok)
			assert.ErrorIs(t, got.err, ErrLockAcquisitionFailed)
			var row Dlock
			require.NoError(t, backend.db.First(&row, "`key` = ?", key).Error)
			assert.Equal(t, nonceB, row.Nonce)
		})
	}
}
