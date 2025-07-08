package store

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/mcuadros/go-defaults"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	ConnMaxLifetime time.Duration `default:"3m"`
	MaxOpenConns    int           `default:"10"`
	MaxIdleConns    int           `default:"10"`

	LogLevel      string        `default:"warn"`
	SlowThreshold time.Duration `default:"200ms"`

	Sqlite *SqliteConfig
	Mysql  *MysqlConfig
}

func NewMemoryConfig() Config {
	config := Config{
		Sqlite: &SqliteConfig{},
	}

	defaults.SetDefaults(&config)

	return config
}

func MustNewConfigFromViper() Config {
	var config Config
	viper.MustUnmarshalKey("store", &config)
	return config
}

func (config *Config) MustOpenOrCreate(tables ...any) *gorm.DB {
	db, err := config.OpenOrCreate(tables...)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to open or create database")
	}

	return db
}

func (config *Config) OpenOrCreate(tables ...any) (*gorm.DB, error) {
	// mysql enabled
	if config.Mysql != nil {
		if err := config.createDatabaseIfAbsent(*config.Mysql); err != nil {
			return nil, errors.WithMessage(err, "Failed to create mysql database if absent")
		}

		return config.openOrCreate(config.Mysql.Open, tables...)
	}

	// sqlite enabled
	if config.Sqlite != nil {
		return config.openOrCreate(config.Sqlite.Open, tables...)
	}

	logrus.Warn("Neither mysql nor sqlite specified, fallback to memory database")

	var memoryConfig SqliteConfig
	defaults.SetDefaults(&memoryConfig)

	return config.openOrCreate(memoryConfig.Open, tables...)
}

func (config *Config) openOrCreate(dialectorFactory func() gorm.Dialector, tables ...any) (*gorm.DB, error) {
	db, err := config.createSession(dialectorFactory())
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create db session")
	}

	sqlDb, err := db.DB()
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to retrieve mysql.DB")
	}

	if err := db.AutoMigrate(tables...); err != nil {
		sqlDb.Close()
		return nil, errors.WithMessage(err, "Failed to auto migrate tables")
	}

	sqlDb.SetConnMaxLifetime(config.ConnMaxLifetime)
	sqlDb.SetMaxOpenConns(config.MaxOpenConns)
	sqlDb.SetMaxIdleConns(config.MaxIdleConns)

	logrus.Debug("MySQL database initialized")

	return db, nil
}

func (config *Config) createSession(dialector gorm.Dialector) (*gorm.DB, error) {
	return gorm.Open(dialector, &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             config.SlowThreshold,
				LogLevel:                  config.getGormLogLevel(),
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		),
	})
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
		logrus.WithField("level", config.LogLevel).Warn("Failed to parse grom log level, fallback to warn")
		return logger.Warn
	}
}

func (config *Config) createDatabaseIfAbsent(mysqlConfig MysqlConfig) error {
	db, err := config.createSession(mysqlConfig.OpenWithoutDB())
	if err != nil {
		return errors.WithMessage(err, "Failed to create mysql db session")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return errors.WithMessage(err, "Failed to retrieve mysql.DB")
	}

	defer sqlDB.Close()

	if _, err = config.Mysql.CreateDatabaseIfAbsent(db); err != nil {
		return errors.WithMessage(err, "Failed to create mysql database if absent")
	}

	return nil
}
