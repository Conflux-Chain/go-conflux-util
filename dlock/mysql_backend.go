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
// Dlock, then call ValidateSchema before using the backend. Use an autocommit
// pool, not a caller-managed transaction: success must include committing writes.
func NewMySQLBackend(db *gorm.DB) *MySQLBackend {
	return &MySQLBackend{db: db}
}

// initializeSchema tolerates concurrent creation, but never migrates existing
// tables. Only known metadata-lock conflicts are retried, with a fixed bound.
func (m *MySQLBackend) initializeSchema(ctx context.Context) error {
	db := m.db.WithContext(ctx)
	for attempt := 0; attempt < 3; attempt++ {
		if db.Migrator().HasTable(&Dlock{}) {
			return m.ValidateSchema(ctx)
		}
		err := db.Migrator().CreateTable(&Dlock{})
		if err == nil {
			return m.ValidateSchema(ctx)
		}
		var mysqlErr *mysql.MySQLError
		if !errors.As(err, &mysqlErr) {
			return err
		}
		switch mysqlErr.Number {
		case 1050: // A competing initializer created the table.
			return m.ValidateSchema(ctx)
		case 1205, 1213: // Lock wait timeout or metadata-lock deadlock.
			if attempt == 2 {
				return errors.Wrap(err, "retry lock table initialization")
			}
		default:
			return err
		}
	}
	return nil
}

// ValidateSchema verifies required columns, defaults and a full unique key
// index. It never modifies existing tables.
func (m *MySQLBackend) ValidateSchema(ctx context.Context) error {
	if m.db.Dialector.Name() != "mysql" {
		return ErrMySQLRequired
	}
	db := m.db.WithContext(ctx)
	schema, table, err := m.schemaTable(ctx)
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
		got := column.Type
		nullable := column.Nullable == "YES"
		required := column.Name != "created_at" && column.Name != "updated_at"
		if strings.ToLower(got) != want || (required && nullable) {
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

// schemaTable accepts GORM's quoted table or schema.table form. Reject custom
// expressions rather than validating metadata for a different table.
func (m *MySQLBackend) schemaTable(ctx context.Context) (schema, table string, err error) {
	table = m.db.Statement.Table
	if table == "" {
		table = (Dlock{}).TableName()
	}
	if expr := m.db.Statement.TableExpr; expr != nil {
		parts := strings.Split(expr.SQL, ".")
		if len(expr.Vars) != 0 || len(parts) > 2 {
			return "", "", ErrIncompatibleSchema
		}
		for i, part := range parts {
			name := strings.ReplaceAll(strings.Trim(part, "`"), "``", "`")
			if name == "" || part != m.db.Statement.Quote(clause.Table{Name: name}) {
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
		err = m.db.WithContext(ctx).Raw("SELECT DATABASE()").Scan(&schema).Error
	}
	return
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
	ctxDb := m.db.WithContext(ctx)
	leaseMillis := lease.Milliseconds()
	if lease%time.Millisecond != 0 {
		leaseMillis++
	}
	expAtGormExpr := gorm.Expr("UTC_TIMESTAMP(3) + INTERVAL ? MICROSECOND", leaseMillis*1000)
	nowTsGormExpr := gorm.Expr("UTC_TIMESTAMP(3)")

	found, err := store.NewStore(ctxDb).Get(&Dlock{}, "`key` = ?", key)
	if err != nil {
		return false, err
	}

	values := map[string]interface{}{
		"nonce":      nonce,
		"expired_at": expAtGormExpr,
		"updated_at": nowTsGormExpr,
	}
	var result *gorm.DB
	if found { // Try to update if expired or nonce matches.
		values["version"] = gorm.Expr("version + 1")
		result = ctxDb.Model(&Dlock{}).
			Where("`key` = ? AND (expired_at <= UTC_TIMESTAMP(3) OR nonce = ?)", key, nonce).
			Updates(values)
	} else { // Attempt to insert a new lock
		values["key"] = key
		values["created_at"] = nowTsGormExpr
		result = ctxDb.Model(&Dlock{}).Create(values)
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
	err = finishAcquisition(ctx, m, key, nonce, start, lease)
	return err == nil, err
}

// Implement DelEntry - release a lock
func (m *MySQLBackend) DelEntry(ctx context.Context, key, nonce string) (bool, error) {
	if err := validateMySQLIdentifiers(key, nonce); err != nil {
		return false, err
	}
	res := m.db.WithContext(ctx).Where("`key` = ? AND nonce = ?", key, nonce).Delete(&Dlock{})
	return res.RowsAffected > 0, res.Error
}
