package store

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MysqlConfig struct {
	Host     string `default:"127.0.0.1:3306"`
	Username string `default:"root"`
	Password string

	Database   string
	AutoCreate bool // creates database if absent
}

func (config *MysqlConfig) OpenWithoutDB() gorm.Dialector {
	// refer to https://github.com/go-sql-driver/mysql#dsn-data-source-name
	dsn := fmt.Sprintf("%v:%v@tcp(%v)/?parseTime=true", config.Username, config.Password, config.Host)

	return mysql.Open(dsn)
}

func (config *MysqlConfig) Open() gorm.Dialector {
	// refer to https://github.com/go-sql-driver/mysql#dsn-data-source-name
	dsn := fmt.Sprintf("%v:%v@tcp(%v)/%v?parseTime=true", config.Username, config.Password, config.Host, config.Database)

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
