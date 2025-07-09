package dlock

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/store"
	"github.com/mcuadros/go-defaults"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var (
	testMysqlBe *MySQLBackend
)

// Please set the following enviroments before running tests:
// `TEST_MYSQL_HOST`: MySQL database host;
// `TEST_MYSQL_USER`: MySQL database username;
// `TEST_MYSQL_PWD`:  MySQL database password;
// `TEST_MYSQL_DB`:   MySQL database database.
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
		return nil
	}

	conf := store.Config{
		Mysql: &store.MysqlConfig{
			Host:     host,
			Database: database,
			Username: os.Getenv("TEST_MYSQL_USER"),
			Password: os.Getenv("TEST_MYSQL_PWD"),
		},
	}

	defaults.SetDefaults(&conf)
	db := conf.MustOpenOrCreate(&Dlock{})
	testMysqlBe = NewMySQLBackend(db)

	return nil
}

func teardown() error {
	if testMysqlBe != nil {
		db, err := testMysqlBe.db.DB()
		if err == nil {
			return db.Close()
		}
	}

	return nil
}

func TestMysqlBackend(t *testing.T) {
	if testMysqlBe == nil {
		t.SkipNow()
	}

	ctx := context.Background()
	lockkey := "shared_key"
	lease := 15 * time.Second

	ok, err := testMysqlBe.WriteEntry(ctx, lockkey, "node0", lease)
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = testMysqlBe.WriteEntry(ctx, lockkey, "node1", lease)
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, err = testMysqlBe.DelEntry(ctx, lockkey, "node1")
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, err = testMysqlBe.DelEntry(ctx, lockkey, "node0")
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestLockManager(t *testing.T) {
	if testMysqlBe == nil {
		t.SkipNow()
	}

	lockman := NewLockManager(testMysqlBe)

	lockKey := "dlock_key"
	lease := 15 * time.Second

	li0 := NewLockIntent(lockKey, "node0", lease)
	err := lockman.Acquire(context.Background(), li0)
	assert.NoError(t, err)

	err = lockman.Acquire(context.Background(), li0)
	assert.NoError(t, err)

	li1 := NewLockIntent(lockKey, "node1", lease)
	err = lockman.Acquire(context.Background(), li1)
	assert.ErrorIs(t, err, ErrLockAcquisitionFailed)

	err = lockman.Release(context.Background(), li1)
	assert.ErrorIs(t, err, ErrLockNotHeld)

	err = lockman.Release(context.Background(), li0)
	assert.NoError(t, err)
}
