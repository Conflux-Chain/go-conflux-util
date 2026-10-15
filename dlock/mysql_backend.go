package dlock

import (
	"context"
	"strings"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/store"
	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const maxKeyBytes = 764

var (
	// Ensure MySQLBackend implements Backend interface.
	_ Backend = (*MySQLBackend)(nil)
)

// Dlock represents a distributed lock.
type Dlock struct {
	ID uint
	// The unique identifier for the lock.
	Key string `gorm:"type:varbinary(764);index:uidx_key,unique;not null"`
	// The secret used to release the lock.
	Nonce string `gorm:"type:longblob;not null"`
	// Version number incremented with each update.
	Version uint `gorm:"default:0;not null"`
	// When the lock expires.
	ExpiredAt time.Time `gorm:"precision:3;not null"`
	// CRUD timestamp.
	CreatedAt, UpdatedAt time.Time `gorm:"precision:3"`
}

// TableName overrides the table name
func (Dlock) TableName() string {
	return "dlocks"
}

// MySQLBackend provides MySQL implementation for Backend interface.
type MySQLBackend struct {
	db *gorm.DB
}

// NewMySQLBackend uses a caller-owned MySQL database. Create or explicitly migrate
// Dlock, then call ValidateSchema before using the backend.
func NewMySQLBackend(db *gorm.DB) *MySQLBackend {
	return &MySQLBackend{db: db}
}

// ValidateSchema verifies identifier types, millisecond timestamps and a full
// unique key index. It never modifies existing tables.
func (m *MySQLBackend) ValidateSchema(ctx context.Context) error {
	if m.db.Dialector.Name() != "mysql" {
		return ErrMySQLRequired
	}
	db := m.db.WithContext(ctx)
	columns, err := db.Migrator().ColumnTypes(&Dlock{})
	if err != nil {
		return err
	}
	expected := map[string]string{
		"key": "varbinary(764)", "nonce": "longblob",
		"expired_at": "datetime(3)", "created_at": "datetime(3)", "updated_at": "datetime(3)",
	}
	for _, column := range columns {
		want, exists := expected[column.Name()]
		if !exists {
			continue
		}
		got, _ := column.ColumnType()
		nullable, _ := column.Nullable()
		required := column.Name() == "key" || column.Name() == "nonce" || column.Name() == "expired_at"
		if strings.ToLower(got) != want || (required && nullable) {
			return errors.Wrapf(ErrIncompatibleSchema, "column %s requires %s", column.Name(), want)
		}
		delete(expected, column.Name())
	}
	if len(expected) != 0 {
		return errors.Wrap(ErrIncompatibleSchema, "missing lock columns")
	}
	table := db.Statement.Table
	if table == "" {
		table = (Dlock{}).TableName()
	}
	// GetIndexes does not expose prefix lengths. Prefix indexes conflate keys;
	// composite indexes can admit duplicate keys. Require a full single-column key.
	var indexes int64
	err = db.Raw(`SELECT COUNT(*) FROM (
		SELECT INDEX_NAME FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
		GROUP BY INDEX_NAME
		HAVING MAX(NON_UNIQUE) = 0 AND COUNT(*) = 1
		AND MAX(COLUMN_NAME) = 'key' AND MAX(SUB_PART) IS NULL
	) AS lock_key_indexes`, table).Scan(&indexes).Error
	if err != nil {
		return err
	}
	if indexes == 0 {
		return errors.Wrap(ErrIncompatibleSchema, "missing full unique key index")
	}
	return nil
}

func validateMySQLIdentifiers(key, nonce string) error {
	if len(key) > maxKeyBytes {
		return ErrInvalidLockKey
	}
	return validateNonce(nonce)
}

// Implement WriteEntry - try to acquire a lock
func (m *MySQLBackend) WriteEntry(
	ctx context.Context, key, nonce string, lease time.Duration) (bool, error) {
	if err := validateMySQLIdentifiers(key, nonce); err != nil {
		return false, err
	}
	if err := validateLease(lease); err != nil {
		return false, err
	}
	ctxDb := m.db.WithContext(ctx)
	leaseMillis := lease.Milliseconds()
	if lease%time.Millisecond != 0 {
		leaseMillis++
	}
	expAtGormExpr := gorm.Expr("UTC_TIMESTAMP(3) + INTERVAL ? MICROSECOND", leaseMillis*1000)
	nowTsGormExpr := gorm.Expr("UTC_TIMESTAMP(3)")
	verGormExpr := gorm.Expr("version + 1")

	found, err := store.NewStore(ctxDb).Get(&Dlock{}, "`key` = ?", key)
	if err != nil {
		return false, err
	}

	if found { // Try to update if expired or nonce matches.
		res := ctxDb.Model(&Dlock{}).
			Where("`key` = ? AND (expired_at <= UTC_TIMESTAMP(3) OR nonce = ?)", key, nonce).
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
			"updated_at": nowTsGormExpr,
		})
		var mysqlErr *mysql.MySQLError
		if errors.Is(res.Error, gorm.ErrDuplicatedKey) || (errors.As(res.Error, &mysqlErr) && mysqlErr.Number == 1062) {
			return false, nil
		}
		return res.Error == nil, res.Error
	}
}

// Implement DelEntry - release a lock
func (m *MySQLBackend) DelEntry(ctx context.Context, key, nonce string) (bool, error) {
	if err := validateMySQLIdentifiers(key, nonce); err != nil {
		return false, err
	}
	res := m.db.WithContext(ctx).Where("`key` = ? AND nonce = ?", key, nonce).Delete(&Dlock{})
	return res.RowsAffected > 0, res.Error
}
