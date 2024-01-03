package source

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go-micro.dev/v4/config"
	"go-micro.dev/v4/config/source"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	// refer to https://github.com/go-sql-driver/mysql#dsn-data-source-name
	testMysqlDsn = "user:password@tcp(127.0.0.1:3306)/dbname?parseTime=true"
	testTableKey = "test.config.json"
	testdb       *gorm.DB
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
	db.Migrator().DropTable(&KvConfig{})

	err = db.Migrator().CreateTable(&KvConfig{})
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create KVConfig table")
	}

	testdb = db
	return nil
}

func teardown() (err error) {
	return testdb.Migrator().DropTable(&KvConfig{})
}

func TestMysqlDataSource(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mysqlDs := NewSource(testdb, testTableKey, &KvConfig{}, WithContext(ctx))
	mysqlDs.Write(&source.ChangeSet{
		Data: []byte("{\"feature\":\"on\"}"),
	})

	err := config.Load(mysqlDs)
	assert.NoError(t, err)

	var featureStatus string
	err = config.Get("feature").Scan(&featureStatus)
	assert.NoError(t, err)
	assert.Equal(t, "on", featureStatus)

	watcher, err := config.Watch("feature")
	assert.NoError(t, err)

	defer watcher.Stop()

	time.AfterFunc(1*time.Second, func() {
		mysqlDs.Write(&source.ChangeSet{
			Data: []byte("{\"feature\":\"off\"}"),
		})
	})

	v, err := watcher.Next()
	assert.NoError(t, err)

	v.Scan(&featureStatus)
	assert.Equal(t, "off", featureStatus)
}
