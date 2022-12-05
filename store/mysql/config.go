package mysql

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	Host     string `default:"127.0.0.1:3306"`
	Username string `default:"root"`
	Password string
	Database string

	ConnMaxLifetime time.Duration `default:"3m"`
	MaxOpenConns    int           `default:"10"`
	MaxIdleConns    int           `default:"10"`

	LogLevel      string        `default:"warn"`
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

func (config *Config) getGormLogLevel() logger.LogLevel {
	switch strings.ToLower(config.LogLevel) {
	case "silent":
		return logger.Silent
	case "info":
		return logger.Info
	case "warn":
		return logger.Warn
	case "error":
		return logger.Error
	default:
		logrus.WithField("level", config.LogLevel).Fatal("Invalid grom log level")
		return logger.Error
	}
}

func (config *Config) MustNewDB(database string) *gorm.DB {
	// create gorm logger by customizing the default logger
	logger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             config.SlowThreshold,     // slow SQL threshold
			LogLevel:                  config.getGormLogLevel(), // log level
			IgnoreRecordNotFoundError: true,                     // never logging on ErrRecordNotFound error, otherwise logs may grow exploded
			Colorful:                  true,                     // use colorful print
		},
	)

	// refer to https://github.com/go-sql-driver/mysql#dsn-data-source-name
	dsn := fmt.Sprintf("%v:%v@tcp(%v)/%v?parseTime=true", config.Username, config.Password, config.Host, database)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger,
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
