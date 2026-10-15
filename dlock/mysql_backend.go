package dlock

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/store"
	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// NewMySQLBackend uses a caller-owned MySQL database. Create or explicitly migrate
// Dlock, then call ValidateSchema before using the backend. Use an autocommit
// pool, not a caller-managed transaction: success must include committing writes.
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

// ValidateSchema verifies required columns, defaults and a full unique key
// index. It never modifies existing tables.
func (backend *MySQLBackend) ValidateSchema(ctx context.Context) error {
	if backend.db.Dialector.Name() != "mysql" {
		return ErrMySQLRequired
	}
	db := backend.db.WithContext(ctx)
	schema, table, err := backend.schemaTable(ctx)
	if err != nil {
		return err
	}
	var columns []struct {
		Name, Type, Nullable, Extra string
		Default                     sql.NullString `gorm:"column:column_default"`
	}
	err = db.Raw(`SELECT COLUMN_NAME AS name, COLUMN_TYPE AS type, IS_NULLABLE AS nullable,
		EXTRA AS extra, COLUMN_DEFAULT AS column_default
		FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?`, schema, table).Scan(&columns).Error
	if err != nil {
		return err
	}
	expected := map[string]string{
		"id": "bigint unsigned", "version": "bigint unsigned",
		"key": "varbinary(764)", "nonce": "longblob",
		"expired_at": "datetime(3)", "created_at": "datetime(3)", "updated_at": "datetime(3)",
	}
	for _, column := range columns {
		want, exists := expected[column.Name]
		if !exists {
			continue
		}
		required := column.Name != "created_at" && column.Name != "updated_at"
		if strings.ToLower(column.Type) != want || (required && column.Nullable == "YES") {
			return errors.Wrapf(ErrIncompatibleSchema, "column %s requires %s", column.Name, want)
		}
		if column.Name == "version" && (!column.Default.Valid || column.Default.String != "0") {
			return errors.Wrap(ErrIncompatibleSchema, "version requires default 0")
		}
		if column.Name == "id" && !strings.Contains(column.Extra, "auto_increment") {
			return errors.Wrap(ErrIncompatibleSchema, "id requires auto_increment")
		}
		delete(expected, column.Name)
	}
	if len(expected) != 0 {
		return errors.Wrap(ErrIncompatibleSchema, "missing lock columns")
	}
	// GetIndexes does not expose prefix lengths. Prefix indexes conflate keys;
	// composite indexes can admit duplicate keys. Require a full single-column key.
	var indexes int64
	err = db.Raw(`SELECT COUNT(*) FROM (
		SELECT INDEX_NAME FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		GROUP BY INDEX_NAME
		HAVING MAX(NON_UNIQUE) = 0 AND COUNT(*) = 1
		AND MAX(COLUMN_NAME) = 'key' AND MAX(SUB_PART) IS NULL
	) AS lock_key_indexes`, schema, table).Scan(&indexes).Error
	if err != nil {
		return err
	}
	if indexes == 0 {
		return errors.Wrap(ErrIncompatibleSchema, "missing full unique key index")
	}
	return nil
}

// initializeSchema tolerates concurrent creation, but never migrates existing
// tables. Only known metadata-lock conflicts are retried, with a fixed bound.
func (backend *MySQLBackend) initializeSchema(ctx context.Context) error {
	const maxAttempts = 3
	db := backend.db.WithContext(ctx)
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if db.Migrator().HasTable(&Dlock{}) {
			break
		}
		err := db.Migrator().CreateTable(&Dlock{})
		if err == nil {
			break
		}
		var mysqlErr *mysql.MySQLError
		if !errors.As(err, &mysqlErr) {
			return err
		}
		switch mysqlErr.Number {
		case 1050: // A competing initializer created the table.
		case 1205, 1213: // Lock wait timeout or metadata-lock deadlock.
			if attempt == maxAttempts-1 {
				return errors.Wrap(err, "retry lock table initialization")
			}
			continue
		default:
			return err
		}
		break
	}
	return backend.ValidateSchema(ctx)
}

// schemaTable accepts GORM's quoted table or schema.table form. Reject custom
// expressions rather than validating metadata for a different table.
func (backend *MySQLBackend) schemaTable(ctx context.Context) (schema, table string, err error) {
	table = backend.db.Statement.Table
	if table == "" {
		table = (Dlock{}).TableName()
	}
	if expr := backend.db.Statement.TableExpr; expr != nil {
		parts := strings.Split(expr.SQL, ".")
		if len(expr.Vars) != 0 || len(parts) > 2 {
			return "", "", ErrIncompatibleSchema
		}
		for i, part := range parts {
			name := strings.ReplaceAll(strings.Trim(part, "`"), "``", "`")
			if name == "" || part != backend.db.Statement.Quote(clause.Table{Name: name}) {
				return "", "", errors.Wrap(ErrIncompatibleSchema, "unsupported table expression")
			}
			parts[i] = name
		}
		table = parts[len(parts)-1]
		if len(parts) == 2 {
			schema = parts[0]
		}
	}
	if schema == "" {
		err = backend.db.WithContext(ctx).Raw("SELECT DATABASE()").Scan(&schema).Error
	}
	return schema, table, err
}

func validateMySQLIdentifiers(key, nonce string) error {
	if len(key) > maxKeyBytes {
		return ErrInvalidLockKey
	}
	return validateNonce(nonce)
}
