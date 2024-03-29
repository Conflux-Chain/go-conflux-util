package dlock

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
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
	Nonce string `gorm:"default:not null"`
	// When the lock expires.
	ExpiredAt time.Time `gorm:"default:not null"`
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
	ctx context.Context, key, nonce string, lease time.Duration) (success bool, err error) {
	if err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		expAtGormExpr := gorm.Expr("NOW() + INTERVAL ? SECOND", lease.Seconds())
		nowTsGormExpr := gorm.Expr("CURRENT_TIMESTAMP")

		// Attempt to insert a new lock
		res := tx.Model(&Dlock{}).Create(map[string]interface{}{
			"key":        key,
			"nonce":      nonce,
			"expired_at": expAtGormExpr,
			"created_at": nowTsGormExpr,
		})

		// Insert successfully?
		if res.Error == nil {
			success = true
			return nil
		}

		if isDuplicateKeyError(res.Error) {
			// Lock exists, try to update if expired or nonce matches.
			res = tx.Model(&Dlock{}).
				Where("`key` = ? AND (expired_at < NOW() OR nonce = ?)", key, nonce).
				Updates(map[string]interface{}{
					"nonce":      nonce,
					"expired_at": expAtGormExpr,
					"updated_at": nowTsGormExpr,
				})

			if res.Error == nil {
				success = res.RowsAffected > 0
				return nil
			}
		}

		return res.Error
	}); err != nil {
		return false, err
	}

	return success, nil
}

// Implement DelEntry - release a lock
func (m *MySQLBackend) DelEntry(ctx context.Context, key, nonce string) (bool, error) {
	res := m.db.WithContext(ctx).Where("`key` = ? AND nonce = ?", key, nonce).Delete(&Dlock{})
	return res.RowsAffected > 0, res.Error
}

// Helper function to check if an error is a duplicate key error
func isDuplicateKeyError(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
