package dlock

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

var (
	// Ensure MySQLBackend implements Backend interface.
	_ Backend = (*MySQLBackend)(nil)
)

// Dlock represents a distributed lock.
type Dlock struct {
	ID uint
	// The unique identifier for the lock.
	Key string `gorm:"index:uidx_key,unique;not null"`
	// The secret used to release the lock.
	Nonce string `gorm:"not null"`
	// Version number incremented with each update.
	Version uint `gorm:"default:0;not null"`
	// When the lock expires.
	ExpiredAt time.Time `gorm:"not null"`
	// CRUD timestamp.
	CreatedAt, UpdatedAt time.Time
}

// TableName overrides the table name
func (Dlock) TableName() string {
	return "dlocks"
}

// MySQLBackend provides MySQL implementation for Backend interface.
type MySQLBackend struct {
	db *gorm.DB
}

// NewMySQLBackend creates a new instance of MySQLBackend.
func NewMySQLBackend(db *gorm.DB) *MySQLBackend {
	return &MySQLBackend{db: db}
}

// Implement WriteEntry - try to acquire a lock
func (m *MySQLBackend) WriteEntry(
	ctx context.Context, key, nonce string, lease time.Duration) (bool, error) {
	ctxDb := m.db.WithContext(ctx)
	expAtGormExpr := gorm.Expr("NOW() + INTERVAL ? SECOND", lease.Seconds())
	nowTsGormExpr := gorm.Expr("CURRENT_TIMESTAMP")
	verGormExpr := gorm.Expr("version + 1")

	err := ctxDb.First(&Dlock{}, "`key` = ?", key).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}

	if err == nil { // Try to update if expired or nonce matches.
		res := ctxDb.WithContext(ctx).Model(&Dlock{}).
			Where("`key` = ? AND (expired_at < NOW() OR nonce = ?)", key, nonce).
			Updates(map[string]interface{}{
				"nonce":      nonce,
				"version":    verGormExpr,
				"expired_at": expAtGormExpr,
				"updated_at": nowTsGormExpr,
			})
		return res.RowsAffected > 0, res.Error
	} else { // Attempt to insert a new lock
		res := ctxDb.Model(&Dlock{}).Create(map[string]interface{}{
			"key":        key,
			"nonce":      nonce,
			"expired_at": expAtGormExpr,
			"created_at": nowTsGormExpr,
		})
		return res.Error == nil, res.Error
	}
}

// Implement DelEntry - release a lock
func (m *MySQLBackend) DelEntry(ctx context.Context, key, nonce string) (bool, error) {
	res := m.db.WithContext(ctx).Where("`key` = ? AND nonce = ?", key, nonce).Delete(&Dlock{})
	return res.RowsAffected > 0, res.Error
}
