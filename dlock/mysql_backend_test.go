package dlock

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/store"
	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func mysqlTestTable(t *testing.T) *MySQLBackend {
	t.Helper()
	name := "dlock_" + mysqlTestKey(t)
	db := testMySQLBackend.db.Table(name).Session(&gorm.Session{})
	require.NoError(t, db.Migrator().CreateTable(&Dlock{}))
	t.Cleanup(func() {
		assert.NoError(t, db.Exec("DROP TABLE IF EXISTS ?", clause.Table{Name: name}).Error)
	})
	return NewMySQLBackend(db)
}

func TestMySQLIdentifiers(t *testing.T) {
	backend := mysqlTestTable(t)
	ctx := context.Background()
	for _, pair := range [][2]string{{"TokenA", "tokena"}, {"space", "space "}, {"café", "cafe"}} {
		key, nonce := testID(t), testID(t)
		ok, err := backend.WriteEntry(ctx, key, nonce+pair[0], time.Minute)
		require.NoError(t, err)
		require.True(t, ok)
		ok, err = backend.WriteEntry(ctx, key, nonce+pair[1], time.Minute)
		require.NoError(t, err)
		assert.False(t, ok)
		ok, err = backend.DelEntry(ctx, key, nonce+pair[1])
		require.NoError(t, err)
		assert.False(t, ok)

		for _, suffix := range pair {
			ok, err = backend.WriteEntry(ctx, nonce+suffix, nonce, time.Minute)
			require.NoError(t, err)
			assert.True(t, ok, "keys must compare by bytes")
		}
	}
	key := strings.Repeat("🔒", 191) // 764 bytes, the previous utf8mb4 capacity.
	ok, err := backend.WriteEntry(ctx, key, testID(t), time.Minute)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestMySQLKeyLengthValidation(t *testing.T) {
	backend := NewMySQLBackend(nil)
	key := strings.Repeat("k", maxKeyBytes+1)
	ok, err := backend.WriteEntry(context.Background(), key, "nonce", time.Second)
	assert.False(t, ok)
	assert.ErrorIs(t, err, ErrInvalidLockKey)
	ok, err = backend.DelEntry(context.Background(), key, "nonce")
	assert.False(t, ok)
	assert.ErrorIs(t, err, ErrInvalidLockKey)
}

func mysqlTestDatabase(t *testing.T) store.Config {
	t.Helper()
	mysqlTestKey(t)
	config := testMySQLConfig()
	config.Mysql.Database = "dlock_" + testID(t)
	name := clause.Table{Name: config.Mysql.Database}
	require.NoError(t, testMySQLBackend.db.Exec("CREATE DATABASE ? CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci", name).Error)
	t.Cleanup(func() { assert.NoError(t, testMySQLBackend.db.Exec("DROP DATABASE ?", name).Error) })
	return config
}

func TestMySQLQualifiedTable(t *testing.T) {
	config := mysqlTestDatabase(t)
	db, err := config.OpenOrCreate(&Dlock{})
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, store.NewStore(db).Close()) })
	target := testMySQLBackend.db.Table(config.Mysql.Database + ".dlocks").Session(&gorm.Session{})
	assertFirstAcquisitionContention(t, NewMySQLBackend(target))
}

func TestMySQLCallerInitialization(t *testing.T) {
	config := mysqlTestDatabase(t)
	db, err := config.OpenOrCreate()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, store.NewStore(db).Close()) })
	ctx := context.Background()
	backend := NewMySQLBackend(db)
	manager := NewLockManager(backend)
	intent := NewLockIntent(testID(t), testID(t), time.Second)
	err = manager.Acquire(ctx, intent)
	var mysqlErr *mysql.MySQLError
	require.ErrorAs(t, err, &mysqlErr)
	assert.EqualValues(t, 1146, mysqlErr.Number)
	assert.False(t, db.Migrator().HasTable(&Dlock{}), "dlock must not create its table")

	readyDB, err := config.OpenOrCreate(&Dlock{})
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, store.NewStore(readyDB).Close()) })
	backend = NewMySQLBackend(readyDB)
	manager = NewLockManager(backend)
	require.NoError(t, manager.Acquire(ctx, intent))
	require.NoError(t, manager.Acquire(ctx, intent)) // Renew using the caller's pool.
	require.NoError(t, manager.Release(ctx, intent))

	// The application owns the pool, including when it is shared with dlock.
	require.NoError(t, store.NewStore(readyDB).Close())
	assert.Error(t, manager.Acquire(ctx, intent))
}

// Pin one connection so MySQL's test clock cannot leak into other pool users.
func withMySQLClock(t *testing.T, backend *MySQLBackend, test func(*MySQLBackend, func(int64))) {
	t.Helper()
	require.NoError(t, backend.db.Connection(func(db *gorm.DB) error {
		db = db.Session(&gorm.Session{})
		defer func() { assert.NoError(t, db.Exec("SET timestamp = DEFAULT").Error) }()
		setTime := func(milliseconds int64) {
			t.Helper()
			timestamp := fmt.Sprintf("%d.%03d", milliseconds/1000, milliseconds%1000)
			// timestamp is formatted solely from integers; MySQL rejects string
			// parameters for this numeric session variable.
			require.NoError(t, db.Exec("SET timestamp = "+timestamp).Error)
		}
		test(NewMySQLBackend(db), setTime)
		return nil
	}))
}

func TestMySQLLeasePrecision(t *testing.T) {
	backend := mysqlTestTable(t)
	withMySQLClock(t, backend, func(backend *MySQLBackend, setTime func(int64)) {
		setTime(1700000000200)
		var now time.Time
		require.NoError(t, backend.db.Raw("SELECT UTC_TIMESTAMP(3)").Scan(&now).Error)
		for _, lease := range []time.Duration{time.Second + time.Nanosecond, 1501500 * time.Microsecond, 1250 * time.Millisecond, time.Duration(1<<63 - 1)} {
			key := testID(t)
			ok, err := backend.WriteEntry(context.Background(), key, testID(t), lease)
			require.NoError(t, err)
			require.True(t, ok)
			var got Dlock
			require.NoError(t, backend.db.First(&got, "`key` = ?", key).Error)
			want := now.Add(lease)
			if lease%time.Millisecond != 0 {
				want = want.Truncate(time.Millisecond).Add(time.Millisecond)
			}
			assert.True(t, got.ExpiredAt.Equal(want), "lease %s: got %s, want %s", lease, got.ExpiredAt, want)
			assert.True(t, got.CreatedAt.Equal(now))
			assert.True(t, got.UpdatedAt.Equal(now))
		}
	})
}

func TestMySQLExpiryBoundary(t *testing.T) {
	backend := mysqlTestTable(t)
	withMySQLClock(t, backend, func(backend *MySQLBackend, setTime func(int64)) {
		const start = int64(1700000000200)
		for _, offset := range []int64{-1, 0, 1} {
			key, a, b := testID(t), testID(t), testID(t)
			setTime(start)
			ok, err := backend.WriteEntry(context.Background(), key, a, time.Second)
			require.NoError(t, err)
			require.True(t, ok)
			setTime(start + 1000 + offset)
			ok, err = backend.WriteEntry(context.Background(), key, b, time.Second)
			require.NoError(t, err)
			assert.Equal(t, offset >= 0, ok, "deadline offset %d ms", offset)
		}
	})
}

func TestMySQLSessionTimeZones(t *testing.T) {
	backend := mysqlTestTable(t)
	for _, zones := range [][2]string{{"+00:00", "+08:00"}, {"+08:00", "-07:00"}} {
		withMySQLClock(t, backend, func(a *MySQLBackend, setA func(int64)) {
			require.NoError(t, a.db.Exec("SET time_zone = ?", zones[0]).Error)
			defer func() { assert.NoError(t, a.db.Exec("SET time_zone = DEFAULT").Error) }()
			withMySQLClock(t, backend, func(b *MySQLBackend, setB func(int64)) {
				require.NoError(t, b.db.Exec("SET time_zone = ?", zones[1]).Error)
				defer func() { assert.NoError(t, b.db.Exec("SET time_zone = DEFAULT").Error) }()
				const start = int64(1700000000200)
				key, nonceA, nonceB := testID(t), testID(t), testID(t)
				setA(start)
				ok, err := a.WriteEntry(context.Background(), key, nonceA, time.Second)
				require.NoError(t, err)
				require.True(t, ok)
				for _, offset := range []int64{0, 999, 1000} {
					setB(start + offset)
					ok, err = b.WriteEntry(context.Background(), key, nonceB, time.Second)
					require.NoError(t, err)
					assert.Equal(t, offset == 1000, ok, "time zones %v, offset %d", zones, offset)
				}
			})
		})
	}
}

func TestMySQLRenewalAndStaleOwner(t *testing.T) {
	backend := mysqlTestTable(t)
	withMySQLClock(t, backend, func(backend *MySQLBackend, setTime func(int64)) {
		ctx := context.Background()
		manager := NewLockManager(backend)
		key := testID(t)
		a := NewLockIntent(key, testID(t), time.Second)
		b := NewLockIntent(key, testID(t), time.Second)
		const start = int64(1700000000200)
		setTime(start)
		require.NoError(t, manager.Acquire(ctx, a))
		setTime(start + 500)
		a.Lease = 2 * time.Second
		require.NoError(t, manager.Acquire(ctx, a))
		for _, offset := range []int64{1000, 2499} {
			setTime(start + offset)
			require.ErrorIs(t, manager.Acquire(ctx, b), ErrLockAcquisitionFailed)
		}
		setTime(start + 2500)
		require.NoError(t, manager.Acquire(ctx, b))
		assert.ErrorIs(t, manager.Release(ctx, a), ErrLockNotHeld)
		assert.ErrorIs(t, manager.Acquire(ctx, a), ErrLockAcquisitionFailed)
		require.NoError(t, manager.Release(ctx, b))
		c := NewLockIntent(key, testID(t), time.Second)
		require.NoError(t, manager.Acquire(ctx, c))
		assert.ErrorIs(t, manager.Release(ctx, b), ErrLockNotHeld)
	})
}

func TestMySQLFirstAcquisitionContention(t *testing.T) {
	for _, translate := range []bool{false, true} {
		t.Run(fmt.Sprintf("translate_errors=%t", translate), func(t *testing.T) {
			backend := mysqlTestTable(t)
			backend.db.Config.TranslateError = translate
			assertFirstAcquisitionContention(t, backend)
		})
	}
}

func assertFirstAcquisitionContention(t *testing.T, backend *MySQLBackend) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	var workers sync.WaitGroup
	defer func() { cancel(); workers.Wait() }()
	queried := make(chan struct{}, 2)
	release := make(chan struct{})
	callback := "dlock_contention_" + testID(t)
	require.NoError(t, backend.db.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == backend.db.Statement.Table && errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			queried <- struct{}{}
			select {
			case <-release:
			case <-ctx.Done():
				tx.AddError(ctx.Err())
			}
		}
	}))
	t.Cleanup(func() { assert.NoError(t, backend.db.Callback().Query().Remove(callback)) })
	results := make(chan error, 2)
	key := testID(t)
	for range 2 {
		intent := NewLockIntent(key, testID(t), time.Minute)
		workers.Add(1)
		go func() {
			defer workers.Done()
			results <- NewLockManager(backend).Acquire(ctx, intent)
		}()
	}
	for range 2 {
		select {
		case <-queried:
		case <-ctx.Done():
			t.Fatal("both acquisitions must observe a missing row before inserting")
		}
	}
	close(release)
	successes := 0
	for range 2 {
		err := <-results
		if err == nil {
			successes++
		} else {
			assert.ErrorIs(t, err, ErrLockAcquisitionFailed)
		}
	}
	assert.Equal(t, 1, successes)
	var rows int64
	require.NoError(t, backend.db.Model(&Dlock{}).Count(&rows).Error)
	assert.EqualValues(t, 1, rows)
}

func TestMySQLCancellation(t *testing.T) {
	backend := mysqlTestTable(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ok, err := backend.WriteEntry(ctx, testID(t), testID(t), time.Second)
	assert.False(t, ok)
	assert.ErrorIs(t, err, context.Canceled)
	ok, err = backend.DelEntry(ctx, testID(t), testID(t))
	assert.False(t, ok)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestMySQLPreservesDatabaseErrors(t *testing.T) {
	for _, tc := range []struct {
		name, ddl string
		seed, del bool
	}{
		{"read", "DROP TABLE ?", false, false},
		{"insert", "ALTER TABLE ? DROP COLUMN created_at", false, false},
		{"update", "ALTER TABLE ? DROP COLUMN expired_at", true, false},
		{"delete", "ALTER TABLE ? DROP COLUMN nonce", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			backend := mysqlTestTable(t)
			ctx := context.Background()
			key, nonce := testID(t), testID(t)
			if tc.seed {
				ok, err := backend.WriteEntry(ctx, key, nonce, time.Minute)
				require.NoError(t, err)
				require.True(t, ok)
			}
			require.NoError(t, backend.db.Exec(tc.ddl, clause.Table{Name: backend.db.Statement.Table}).Error)
			var ok bool
			var err error
			if tc.del {
				ok, err = backend.DelEntry(ctx, key, nonce)
			} else {
				ok, err = backend.WriteEntry(ctx, key, nonce, time.Minute)
			}
			assert.False(t, ok)
			var mysqlErr *mysql.MySQLError
			assert.ErrorAs(t, err, &mysqlErr)
			assert.NotErrorIs(t, err, ErrLockAcquisitionFailed)
		})
	}
}
