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

var testMySQLBackend *MySQLBackend

// Set the following environment variables before running tests:
// `TEST_MYSQL_HOST`: MySQL database host;
// `TEST_MYSQL_USER`: MySQL database username;
// `TEST_MYSQL_PWD`:  MySQL database password;
// `TEST_MYSQL_DB`:   dedicated MySQL test database, never a production database.
// `TEST_MYSQL_REQUIRED=1` fails setup instead of skipping if configuration is absent.
func TestMain(m *testing.M) {
	if err := setup(); err != nil {
		panic(errors.WithMessage(err, "failed to set up"))
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

	config := testMySQLConfig()
	db, err := config.OpenOrCreate(&Dlock{})
	if err != nil {
		return err
	}
	testMySQLBackend = NewMySQLBackend(db)
	return nil
}

func testMySQLConfig() store.Config {
	config := store.Config{
		Mysql: &store.MysqlConfig{
			Host:     os.Getenv("TEST_MYSQL_HOST"),
			Database: os.Getenv("TEST_MYSQL_DB"),
			Username: os.Getenv("TEST_MYSQL_USER"),
			Password: os.Getenv("TEST_MYSQL_PWD"),
		},
	}

	defaults.SetDefaults(&config)
	return config
}

func teardown() error {
	if testMySQLBackend != nil {
		return store.NewStore(testMySQLBackend.db).Close()
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
	if testMySQLBackend == nil {
		t.Skip("set TEST_MYSQL_HOST and TEST_MYSQL_DB to run MySQL integration tests")
	}
	key := testID(t)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		assert.NoError(t, testMySQLBackend.db.WithContext(ctx).Where("`key` = ?", key).Delete(&Dlock{}).Error)
	})
	return key
}

func TestMySQLBackend(t *testing.T) {
	ctx := context.Background()
	lockKey := mysqlTestKey(t)
	nonce0, nonce1 := testID(t), testID(t)
	lease := 15 * time.Second

	ok, err := testMySQLBackend.WriteEntry(ctx, lockKey, nonce0, lease)
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = testMySQLBackend.WriteEntry(ctx, lockKey, nonce1, lease)
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, err = testMySQLBackend.DelEntry(ctx, lockKey, nonce1)
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, err = testMySQLBackend.DelEntry(ctx, lockKey, nonce0)
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestLockManager(t *testing.T) {
	lockKey := mysqlTestKey(t)
	manager := NewLockManager(testMySQLBackend)

	lease := 15 * time.Second

	intent0 := NewLockIntent(lockKey, testID(t), lease)
	err := manager.Acquire(context.Background(), intent0)
	assert.NoError(t, err)

	err = manager.Acquire(context.Background(), intent0)
	assert.NoError(t, err)

	intent1 := NewLockIntent(lockKey, testID(t), lease)
	err = manager.Acquire(context.Background(), intent1)
	assert.ErrorIs(t, err, ErrLockAcquisitionFailed)

	err = manager.Release(context.Background(), intent1)
	assert.ErrorIs(t, err, ErrLockNotHeld)

	err = manager.Release(context.Background(), intent0)
	assert.NoError(t, err)
}
