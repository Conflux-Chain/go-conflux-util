package store

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MysqlConfig struct {
	Host       string `default:"127.0.0.1:3306"`
	Username   string `default:"root"`
	Password   string
	Database   string
	AutoCreate bool   // indicates whether to create database if absent
	Location   string // time.Location name, e.g. "UTC (default)", "Local" or "Asia%2FShanghai", where "%2F" escapes "/"
}

// Open opens a mysql gorm dialector. If the `dbName` not specified, `config.Database` will be used.
//
// If you do not want to preselect a database, use empty string for `dbName`.
func (config *MysqlConfig) Open(dbName ...string) gorm.Dialector {
	database := config.Database
	if len(dbName) > 0 {
		database = dbName[0]
	}

	// refer to https://github.com/go-sql-driver/mysql#dsn-data-source-name
	dsn := fmt.Sprintf("%v:%v@tcp(%v)/%v?parseTime=true", config.Username, config.Password, config.Host, database)

	if len(config.Location) > 0 {
		dsn += fmt.Sprintf("&loc=%v", config.Location)
	}

	return mysql.Open(dsn)
}

func (config *MysqlConfig) CreateDatabaseIfAbsent(db *gorm.DB) (bool, error) {
	if !config.AutoCreate {
		return false, nil
	}

	var databases []string
	if err := db.Raw(fmt.Sprintf("SHOW DATABASES LIKE '%v'", config.Database)).Find(&databases).Error; err != nil {
		return false, errors.WithMessage(err, "Failed to query databases")
	}

	if len(databases) > 0 {
		return false, nil
	}

	dbCreateSql := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS %v CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci",
		config.Database,
	)
	if err := db.Exec(dbCreateSql).Error; err != nil {
		return false, errors.WithMessage(err, "Failed to create database")
	}

	logrus.WithField("name", config.Database).Info("Create mysql database for the first time")

	return true, nil
}
