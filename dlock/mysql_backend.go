package dlock

import (
	"context"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/store"
	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const maxKeyBytes = 764

var _ Backend = (*MySQLBackend)(nil)

// Dlock represents a distributed lock.
type Dlock struct {
	ID uint
	// The unique identifier for the lock.
	Key string `gorm:"type:varbinary(764);index:uidx_key,unique;not null"`
	// The ownership token used to renew or release the lock.
	Nonce string `gorm:"type:longblob;not null"`
	// Version number incremented with each update.
	Version uint `gorm:"default:0;not null"`
	// When the lock expires.
	ExpiredAt time.Time `gorm:"precision:3;not null"`
	// Creation and last-update timestamps.
	CreatedAt, UpdatedAt time.Time `gorm:"precision:3"`
}

// TableName returns the lock table name.
func (Dlock) TableName() string {
	return "dlocks"
}

// MySQLBackend provides MySQL implementation for Backend interface.
type MySQLBackend struct {
	db *gorm.DB
}

// NewMySQLBackend uses a caller-owned MySQL database and does not create tables.
// Initialize the table from the Dlock model before using the backend.
// Use an autocommit pool, not a caller-managed transaction: success must include
// committing writes.
func NewMySQLBackend(db *gorm.DB) *MySQLBackend {
	return &MySQLBackend{db: db}
}

// WriteEntry attempts to acquire or renew a lock with a usable lease.
func (backend *MySQLBackend) WriteEntry(
	ctx context.Context, key, nonce string, lease time.Duration) (bool, error) {
	start := time.Now()
	if err := validateMySQLIdentifiers(key, nonce); err != nil {
		return false, err
	}
	if err := validateLease(lease); err != nil {
		return false, err
	}
	if time.Since(start) >= lease-leasePrecision {
		return false, ErrLockAcquisitionFailed
	}
	db := backend.db.WithContext(ctx)
	leaseMillis := lease.Milliseconds()
	if lease%time.Millisecond != 0 {
		leaseMillis++
	}
	expiresAt := gorm.Expr("UTC_TIMESTAMP(3) + INTERVAL ? MICROSECOND", leaseMillis*1000)
	now := gorm.Expr("UTC_TIMESTAMP(3)")

	found, err := store.NewStore(db).Get(&Dlock{}, "`key` = ?", key)
	if err != nil {
		return false, err
	}

	values := map[string]interface{}{
		"nonce":      nonce,
		"expired_at": expiresAt,
		"updated_at": now,
	}
	var result *gorm.DB
	if found { // Try to update if expired or nonce matches.
		values["version"] = gorm.Expr("version + 1")
		result = db.Model(&Dlock{}).
			Where("`key` = ? AND (expired_at <= UTC_TIMESTAMP(3) OR nonce = ?)", key, nonce).
			Updates(values)
	} else { // Attempt to insert a new lock
		values["key"] = key
		values["created_at"] = now
		result = db.Model(&Dlock{}).Create(values)
		var mysqlErr *mysql.MySQLError
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) || (errors.As(result.Error, &mysqlErr) && mysqlErr.Number == 1062) {
			return false, nil
		}
	}
	if result.Error != nil {
		return false, result.Error
	}
	if found && result.RowsAffected == 0 {
		return false, nil
	}
	err = finishAcquisition(ctx, backend, key, nonce, start, lease)
	return err == nil, err
}

// DelEntry releases a lock only if its key and nonce match.
func (backend *MySQLBackend) DelEntry(ctx context.Context, key, nonce string) (bool, error) {
	if err := validateMySQLIdentifiers(key, nonce); err != nil {
		return false, err
	}
	result := backend.db.WithContext(ctx).Where("`key` = ? AND nonce = ?", key, nonce).Delete(&Dlock{})
	return result.RowsAffected > 0, result.Error
}

func validateMySQLIdentifiers(key, nonce string) error {
	if len(key) > maxKeyBytes {
		return ErrInvalidLockKey
	}
	return validateNonce(nonce)
}
