package store

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type SqliteConfig struct {
	Path string `default:":memory:"`
}

func (config *SqliteConfig) Open() gorm.Dialector {
	return sqlite.Open(config.Path)
}
