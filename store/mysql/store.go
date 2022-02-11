package mysql

import (
	"time"

	"github.com/Conflux-Chain/go-conflux-util/store"
	viperutil "github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// MysqlConfig represents the MySQL configurations to open a database instance.
type MysqlConfig struct {
	// connection DSN
	Dsn string
	// connection max life time
	ConnMaxLifetime time.Duration `default:"3ms"`
	// max open connections
	MaxOpenConns int `default:"10"`
	// max idle connections
	MaxIdleConns int `default:"10"`
}

// MysqlStore persists data in MySQL db.
type MysqlStore struct {
	*gorm.DB
}

// MustNewStoreFromViper creates an instance of MySQL store from viper
// or panic on error.
func MustNewStoreFromViper() *MysqlStore {
	store, err := NewStoreFromViper()
	if err != nil {
		logrus.Fatal("Failed to create MySQL store from viper")
	}

	return store
}

// NewStoreFromViper creates an instance of MySQL store from viper.
func NewStoreFromViper() (*MysqlStore, error) {
	var config MysqlConfig

	err := viperutil.UnmarshalKey("store.mysql", &config)
	if err != nil {
		err = errors.WithMessage(err, "failed to unmarshal config from viper")
		return nil, err
	}

	db, err := gorm.Open(mysql.Open(config.Dsn))
	if err != nil {
		err = errors.WithMessage(err, "failed to open MySQL db")
		return nil, err
	}

	innerdb, err := db.DB()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to init MySQL db")
	}

	innerdb.SetConnMaxLifetime(config.ConnMaxLifetime)
	innerdb.SetMaxOpenConns(config.MaxOpenConns)
	innerdb.SetMaxIdleConns(config.MaxIdleConns)

	return &MysqlStore{DB: db}, nil
}

// EnsureTableExistence ensures table existence by creating table schema
// if table not existed yet.
//
// Note that it never updates table schema if table already exists.
func (ms *MysqlStore) EnsureTableExistence(tables ...interface{}) error {
	var newTables []interface{}
	for _, tbl := range tables {
		if !ms.Migrator().HasTable(tbl) {
			newTables = append(newTables, tbl)
		}
	}

	if len(newTables) == 0 {
		return nil
	}

	err := ms.Migrator().CreateTable(tables...)
	return errors.WithMessage(err, "failed to create table schemas")
}

// IsRecordNotFound implements Store interface method.
func (ms *MysqlStore) IsRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound) || err == store.ErrRecordNotFound
}

// Close implements Store interface method.
func (ms *MysqlStore) Close() error {
	if mysqlDb, err := ms.DB.DB(); err != nil {
		return err
	} else {
		return mysqlDb.Close()
	}
}
