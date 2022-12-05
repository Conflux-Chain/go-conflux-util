package mysql

import (
	"fmt"
	stdLog "log"
	"os"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

type Config struct {
	Host     string `default:"127.0.0.1:3306"`
	Username string `default:"root"`
	Password string
	Database string

	ConnMaxLifetime time.Duration `default:"3m"`
	MaxOpenConns    int           `default:"10"`
	MaxIdleConns    int           `default:"10"`

	SlowThreshold time.Duration `default:"200ms"`
}

func MustNewConfigFromViper() Config {
	var config Config
	viper.MustUnmarshalKey("store.mysql", &config)
	return config
}

// MustOpenOrCreate creates an instance of store or exits on any error.
func (config *Config) MustOpenOrCreate(tables ...interface{}) *gorm.DB {
	config.MustCreateDatabaseIfAbsent()

	db := config.MustNewDB(config.Database)

	var currentTables []string
	if err := db.Raw("SHOW TABLES").Find(&currentTables).Error; err != nil {
		logrus.WithError(err).Fatal("Failed to query tables")
	}

	if len(currentTables) == 0 {
		if err := db.Migrator().CreateTable(tables...); err != nil {
			logrus.WithError(err).Fatal("Failed to create database tables")
		}
	}

	if sqlDb, err := db.DB(); err != nil {
		logrus.WithError(err).Fatal("Failed to init mysql db")
	} else {
		sqlDb.SetConnMaxLifetime(config.ConnMaxLifetime)
		sqlDb.SetMaxOpenConns(config.MaxOpenConns)
		sqlDb.SetMaxIdleConns(config.MaxIdleConns)
	}

	logrus.Debug("MySQL database initialized")

	return db
}

func (config *Config) MustNewDB(database string) *gorm.DB {
	gLogLevel := gormLogger.Error
	if logrus.IsLevelEnabled(logrus.TraceLevel) {
		gLogLevel = gormLogger.Info
	} else if logrus.IsLevelEnabled(logrus.WarnLevel) {
		gLogLevel = gormLogger.Warn
	}

	// create gorm logger by customizing the default logger
	gLogger := gormLogger.New(
		stdLog.New(os.Stdout, "\r\n", stdLog.LstdFlags), // io writer
		gormLogger.Config{
			SlowThreshold:             config.SlowThreshold, // slow SQL threshold
			LogLevel:                  gLogLevel,            // log level
			IgnoreRecordNotFoundError: true,                 // never logging on ErrRecordNotFound error, otherwise logs may grow exploded
			Colorful:                  true,                 // use colorful print
		},
	)

	// refer to https://github.com/go-sql-driver/mysql#dsn-data-source-name
	dsn := fmt.Sprintf("%v:%v@tcp(%v)/%v?parseTime=true", config.Username, config.Password, config.Host, database)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gLogger,
	})

	if err != nil {
		logrus.WithError(err).Fatal("Failed to open mysql")
	}

	return db
}

func (config *Config) MustCreateDatabaseIfAbsent() bool {
	db := config.MustNewDB("")
	if mysqlDb, err := db.DB(); err != nil {
		return false
	} else {
		defer mysqlDb.Close()
	}

	var databases []string
	if err := db.Raw(fmt.Sprintf("SHOW DATABASES LIKE '%v'", config.Database)).Find(&databases).Error; err != nil {
		logrus.WithError(err).Fatal("Failed to query databases")
	}

	if len(databases) > 0 {
		return false
	}

	dbCreateSql := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS %v CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci",
		config.Database,
	)
	if err := db.Exec(dbCreateSql).Error; err != nil {
		logrus.WithError(err).Fatal("Failed to create database")
	}

	logrus.WithField("name", config.Database).Info("Create database for the first time")

	return true
}
