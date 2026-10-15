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
	require.NoError(t, backend.ValidateSchema(ctx))
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

func TestMySQLSchemaValidation(t *testing.T) {
	for _, alteration := range []string{
		"MODIFY `key` varchar(191) COLLATE utf8mb4_general_ci NOT NULL",
		"MODIFY nonce longtext COLLATE utf8mb4_general_ci NOT NULL",
		"MODIFY nonce longblob NULL",
		"MODIFY expired_at datetime(0) NOT NULL",
		"MODIFY created_at datetime(6)",
		"MODIFY updated_at datetime(0)",
		"DROP INDEX uidx_key",
		"DROP INDEX uidx_key, ADD UNIQUE INDEX uidx_key (`key`(10))",
		"DROP INDEX uidx_key, ADD UNIQUE INDEX uidx_key (`key`, version)",
	} {
		t.Run(alteration, func(t *testing.T) {
			backend := mysqlTestTable(t)
			require.NoError(t, backend.db.Exec("ALTER TABLE ? "+alteration, clause.Table{Name: backend.db.Statement.Table}).Error)
			assert.ErrorIs(t, backend.ValidateSchema(context.Background()), ErrIncompatibleSchema)
		})
	}
}

func TestMySQLRequiredColumns(t *testing.T) {
	for _, alteration := range []string{
		"DROP COLUMN version",
		"MODIFY version varchar(32) NOT NULL DEFAULT '0'",
		"MODIFY version int unsigned NOT NULL DEFAULT 0",
		"MODIFY version bigint unsigned NULL DEFAULT 0",
		"ALTER version DROP DEFAULT",
		"ALTER version SET DEFAULT 1",
		"DROP COLUMN id",
		"MODIFY id int unsigned NOT NULL AUTO_INCREMENT",
		"MODIFY id bigint unsigned NOT NULL",
	} {
		t.Run(alteration, func(t *testing.T) {
			backend := mysqlTestTable(t)
			require.NoError(t, backend.db.Exec("ALTER TABLE ? "+alteration, clause.Table{Name: backend.db.Statement.Table}).Error)
			assert.ErrorIs(t, backend.ValidateSchema(context.Background()), ErrIncompatibleSchema)
		})
	}
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

func TestMySQLQualifiedSchema(t *testing.T) {
	original := mysqlTestTable(t)
	config := mysqlTestDatabase(t)
	table := original.db.Statement.Table
	target := original.db.Table(config.Mysql.Database + "." + table).Session(&gorm.Session{})
	require.NoError(t, target.Migrator().CreateTable(&Dlock{}))
	backend := NewMySQLBackend(target)
	ctx := context.Background()
	require.NoError(t, backend.ValidateSchema(ctx))
	require.NoError(t, target.Exec("ALTER TABLE ? DROP INDEX uidx_key", clause.Table{Name: config.Mysql.Database + "." + table}).Error)
	assert.ErrorIs(t, backend.ValidateSchema(ctx), ErrIncompatibleSchema)
	require.NoError(t, target.Exec("ALTER TABLE ? ADD UNIQUE INDEX uidx_key (`key`)", clause.Table{Name: config.Mysql.Database + "." + table}).Error)
	require.NoError(t, original.db.Exec("ALTER TABLE ? DROP INDEX uidx_key, MODIFY nonce longtext", clause.Table{Name: table}).Error)
	require.NoError(t, backend.ValidateSchema(ctx), "default schema must not affect target validation")
	require.NoError(t, original.db.Exec("DROP TABLE ?", clause.Table{Name: table}).Error)
	require.NoError(t, backend.ValidateSchema(ctx), "no default table is required")
	assertFirstAcquisitionContention(t, backend)

	// Column checks must resolve the same target as index checks.
	require.NoError(t, target.Exec("ALTER TABLE ? MODIFY nonce longtext", clause.Table{Name: config.Mysql.Database + "." + table}).Error)
	assert.ErrorIs(t, backend.ValidateSchema(ctx), ErrIncompatibleSchema)
	assert.ErrorIs(t, NewMySQLBackend(original.db.Table(table+" AS alias")).ValidateSchema(ctx), ErrIncompatibleSchema)
}

func TestMySQLExplicitMigration(t *testing.T) {
	config := mysqlTestDatabase(t)
	manager, err := NewLockManagerFromConfig(config) // Creates a new, compatible table.
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, manager.Close()) })
	db := manager.backend.(*MySQLBackend).db
	row := Dlock{Key: "LegacyKey", Nonce: "TokenA", Version: 7, ExpiredAt: time.Now().Add(time.Hour)}
	require.NoError(t, db.Create(&row).Error)
	require.NoError(t, db.Exec("ALTER TABLE dlocks MODIFY `key` varchar(191) NOT NULL, MODIFY nonce longtext NOT NULL").Error)

	countPools := func() int64 {
		var n int64
		require.NoError(t, db.Raw("SELECT COUNT(*) FROM information_schema.PROCESSLIST WHERE DB = DATABASE()").Scan(&n).Error)
		return n
	}
	before := countPools()
	failed, err := NewLockManagerFromConfig(config)
	assert.Nil(t, failed)
	assert.ErrorIs(t, err, ErrIncompatibleSchema)
	assert.Eventually(t, func() bool { return countPools() == before }, time.Second, 10*time.Millisecond)
	var columnType string
	require.NoError(t, db.Raw("SELECT DATA_TYPE FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'dlocks' AND COLUMN_NAME = 'key'").Scan(&columnType).Error)
	assert.Equal(t, "varchar", columnType, "initialization must not migrate existing tables")

	require.NoError(t, db.Exec(`ALTER TABLE dlocks
		MODIFY `+"`key`"+` VARBINARY(764) NOT NULL,
		MODIFY nonce LONGBLOB NOT NULL,
		MODIFY expired_at DATETIME(3) NOT NULL,
		MODIFY created_at DATETIME(3) NULL,
		MODIFY updated_at DATETIME(3) NULL`).Error)
	reopened, err := NewLockManagerFromConfig(config)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, reopened.Close()) })
	var got Dlock
	require.NoError(t, db.First(&got, row.ID).Error)
	assert.Equal(t, row.Key, got.Key)
	assert.Equal(t, row.Nonce, got.Nonce)
	assert.Equal(t, row.Version, got.Version)

	// Caller-owned databases remain open even after validation failures.
	pool, err := db.DB()
	require.NoError(t, err)
	assert.NoError(t, pool.Ping())
}

func TestSchemaRejectsSQLite(t *testing.T) {
	config := store.NewMemoryConfig()
	db, err := config.OpenOrCreate()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, store.NewStore(db).Close()) })
	assert.ErrorIs(t, NewMySQLBackend(db).ValidateSchema(context.Background()), ErrMySQLRequired)
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
