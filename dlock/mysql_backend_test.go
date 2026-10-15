package dlock

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

func mysqlTestTable(t *testing.T) *MySQLBackend {
	t.Helper()
	name := "dlock_" + mysqlTestKey(t)
	db := testMysqlBe.db.Table(name)
	require.NoError(t, db.Migrator().CreateTable(&Dlock{}))
	t.Cleanup(func() {
		assert.NoError(t, db.Exec("DROP TABLE IF EXISTS ?", clause.Table{Name: name}).Error)
	})
	return NewMySQLBackend(db)
}

func TestMySQLIdentifiers(t *testing.T) {
	be := mysqlTestTable(t)
	ctx := context.Background()
	require.NoError(t, be.ValidateSchema(ctx))
	for _, pair := range [][2]string{{"TokenA", "tokena"}, {"space", "space "}, {"café", "cafe"}} {
		key, nonce := testID(t), testID(t)
		ok, err := be.WriteEntry(ctx, key, nonce+pair[0], time.Minute)
		require.NoError(t, err)
		require.True(t, ok)
		ok, err = be.WriteEntry(ctx, key, nonce+pair[1], time.Minute)
		require.NoError(t, err)
		assert.False(t, ok)
		ok, err = be.DelEntry(ctx, key, nonce+pair[1])
		require.NoError(t, err)
		assert.False(t, ok)

		for _, suffix := range pair {
			ok, err = be.WriteEntry(ctx, nonce+suffix, nonce, time.Minute)
			require.NoError(t, err)
			assert.True(t, ok, "keys must compare by bytes")
		}
	}
	key := strings.Repeat("🔒", 191) // 764 bytes, the previous utf8mb4 capacity.
	ok, err := be.WriteEntry(ctx, key, testID(t), time.Minute)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestMySQLKeyLengthValidation(t *testing.T) {
	be := NewMySQLBackend(nil)
	key := strings.Repeat("k", maxKeyBytes+1)
	ok, err := be.WriteEntry(context.Background(), key, "nonce", time.Second)
	assert.False(t, ok)
	assert.ErrorIs(t, err, ErrInvalidLockKey)
	ok, err = be.DelEntry(context.Background(), key, "nonce")
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
			be := mysqlTestTable(t)
			require.NoError(t, be.db.Exec("ALTER TABLE ? "+alteration, clause.Table{Name: be.db.Statement.Table}).Error)
			assert.ErrorIs(t, be.ValidateSchema(context.Background()), ErrIncompatibleSchema)
		})
	}
}

func TestMySQLExplicitMigration(t *testing.T) {
	mysqlTestKey(t)
	conf := testMySQLConfig()
	conf.Mysql.Database = "dlock_" + testID(t)
	name := clause.Table{Name: conf.Mysql.Database}
	require.NoError(t, testMysqlBe.db.Exec("CREATE DATABASE ? CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci", name).Error)
	t.Cleanup(func() { assert.NoError(t, testMysqlBe.db.Exec("DROP DATABASE ?", name).Error) })
	manager, err := NewLockManagerFromConfig(conf) // Creates a new, compatible table.
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
	failed, err := NewLockManagerFromConfig(conf)
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
	reopened, err := NewLockManagerFromConfig(conf)
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
	conf := store.NewMemoryConfig()
	db, err := conf.OpenOrCreate()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, store.NewStore(db).Close()) })
	assert.ErrorIs(t, NewMySQLBackend(db).ValidateSchema(context.Background()), ErrMySQLRequired)
}
