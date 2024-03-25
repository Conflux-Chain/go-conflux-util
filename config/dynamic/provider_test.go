package dynamic_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/config/dynamic"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	testdb *gorm.DB
)

const (
	// refer to https://github.com/go-sql-driver/mysql#dsn-data-source-name
	testMysqlDsn        = "user:password@tcp(127.0.0.1:3306)/dbname?parseTime=true"
	testConfKey         = "test.config.json"
	testMonitorInterval = time.Second
)

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
	db, err := gorm.Open(mysql.Open(testMysqlDsn))
	if err != nil {
		return errors.WithMessage(err, "failed to open MySQL database")
	}

	// in case table not cleaned up after tested before
	db.Migrator().DropTable(&dynamic.KvConfig{})

	err = db.Migrator().CreateTable(&dynamic.KvConfig{})
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create KVConfig table")
	}

	testdb = db
	return nil
}

func teardown() (err error) {
	return testdb.Migrator().DropTable(&dynamic.KvConfig{})
}

func TestKvConfigProvider(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cp := dynamic.NewKvConfigProvider(testdb, dynamic.KvConfig{}, testMonitorInterval)
	err := cp.Set(testConfKey, "{\"feature\":\"off\"}")
	assert.NoError(t, err)

	go cp.Watch(ctx, testConfKey)

	data, err := cp.Get(testConfKey)
	assert.NoError(t, err)

	configMap := make(map[string]string)
	assert.NoError(t, json.Unmarshal([]byte(data), &configMap))
	assert.Equal(t, "off", configMap["feature"])

	cp.RegisterCallback("watcher", func(kp *dynamic.KvPair) {
		configMap := make(map[string]string)
		assert.NoError(t, json.Unmarshal([]byte(kp.Value), &configMap))
		assert.Equal(t, "on", configMap["feature"])
	})

	err = cp.Set(testConfKey, "{\"feature\":\"on\"}")
	assert.NoError(t, err)

	time.Sleep(testMonitorInterval)
}
