package dlock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/store"
	"github.com/mcuadros/go-defaults"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testMysqlBe      *MySQLBackend
	testMysqlManager *LockManager
)

// Please set the following environments before running tests:
// `TEST_MYSQL_HOST`: MySQL database host;
// `TEST_MYSQL_USER`: MySQL database username;
// `TEST_MYSQL_PWD`:  MySQL database password;
// `TEST_MYSQL_DB`:   dedicated MySQL test database, never a production database.
// `TEST_MYSQL_REQUIRED=1` fails setup instead of skipping if configuration is absent.
func TestMain(m *testing.M) {
	if err := setup(); err != nil {
		panic(errors.WithMessage(err, "failed to setup"))
	}

	code := m.Run()

	if err := teardown(); err != nil {
		panic(errors.WithMessage(err, "failed to tear down"))
	}

	os.Exit(code)
}

func setup() error {
	host := os.Getenv("TEST_MYSQL_HOST")
	database := os.Getenv("TEST_MYSQL_DB")
	if len(host) == 0 || len(database) == 0 {
		if os.Getenv("TEST_MYSQL_REQUIRED") == "1" || host != "" || database != "" {
			return errors.New("MySQL tests require both TEST_MYSQL_HOST and TEST_MYSQL_DB")
		}
		return nil
	}

	conf := testMySQLConfig()
	var err error
	testMysqlManager, err = NewLockManagerFromConfig(conf)
	if err == nil {
		testMysqlBe = testMysqlManager.backend.(*MySQLBackend)
	}
	return err
}

func testMySQLConfig() store.Config {
	conf := store.Config{
		Mysql: &store.MysqlConfig{
			Host:     os.Getenv("TEST_MYSQL_HOST"),
			Database: os.Getenv("TEST_MYSQL_DB"),
			Username: os.Getenv("TEST_MYSQL_USER"),
			Password: os.Getenv("TEST_MYSQL_PWD"),
		},
	}

	defaults.SetDefaults(&conf)
	return conf
}

func teardown() error {
	if testMysqlManager != nil {
		return testMysqlManager.Close()
	}

	return nil
}

func testID(t *testing.T) string {
	t.Helper()
	var id [16]byte
	_, err := rand.Read(id[:])
	require.NoError(t, err)
	return hex.EncodeToString(id[:])
}

func mysqlTestKey(t *testing.T) string {
	t.Helper()
	if testMysqlBe == nil {
		t.Skip("set TEST_MYSQL_HOST and TEST_MYSQL_DB to run MySQL integration tests")
	}
	key := testID(t)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		assert.NoError(t, testMysqlBe.db.WithContext(ctx).Where("`key` = ?", key).Delete(&Dlock{}).Error)
	})
	return key
}

func TestMysqlBackend(t *testing.T) {
	ctx := context.Background()
	lockkey := mysqlTestKey(t)
	nonce0, nonce1 := testID(t), testID(t)
	lease := 15 * time.Second

	ok, err := testMysqlBe.WriteEntry(ctx, lockkey, nonce0, lease)
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = testMysqlBe.WriteEntry(ctx, lockkey, nonce1, lease)
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, err = testMysqlBe.DelEntry(ctx, lockkey, nonce1)
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, err = testMysqlBe.DelEntry(ctx, lockkey, nonce0)
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestLockManager(t *testing.T) {
	lockKey := mysqlTestKey(t)
	lockman := NewLockManager(testMysqlBe)

	lease := 15 * time.Second

	li0 := NewLockIntent(lockKey, testID(t), lease)
	err := lockman.Acquire(context.Background(), li0)
	assert.NoError(t, err)

	err = lockman.Acquire(context.Background(), li0)
	assert.NoError(t, err)

	li1 := NewLockIntent(lockKey, testID(t), lease)
	err = lockman.Acquire(context.Background(), li1)
	assert.ErrorIs(t, err, ErrLockAcquisitionFailed)

	err = lockman.Release(context.Background(), li1)
	assert.ErrorIs(t, err, ErrLockNotHeld)

	err = lockman.Release(context.Background(), li0)
	assert.NoError(t, err)
}

func TestManagerPoolOwnership(t *testing.T) {
	key := mysqlTestKey(t)
	conf := testMySQLConfig()
	conf.Sqlite = &store.SqliteConfig{} // MySQL must still take precedence.
	manager, err := NewLockManagerFromConfig(conf)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, manager.Close()) })
	pool, err := manager.backend.(*MySQLBackend).db.DB()
	require.NoError(t, err)
	require.NoError(t, manager.Acquire(context.Background(), NewLockIntent(key, testID(t), time.Second)))
	assert.NoError(t, manager.Close())
	assert.NoError(t, manager.Close())
	assert.Error(t, pool.Ping())

	external := NewLockManager(testMysqlBe)
	assert.NoError(t, external.Close())
	assert.NoError(t, external.Close())
	pool, err = testMysqlBe.db.DB()
	require.NoError(t, err)
	assert.NoError(t, pool.Ping())
}
