package dlock

import (
	"context"
	"database/sql"
	"errors"
	"time"

	driver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// MySQLBackend bypasses GORM statements, callbacks, logging and routing.
// Only independent *sql.DB pools are supported, never transactions/fixed conns.
type MySQLBackend struct{ db *sql.DB }

func NewMySQLBackend(db *gorm.DB) (*MySQLBackend, error) {
	if db == nil || db.Dialector == nil || db.Dialector.Name() != "mysql" || db.Statement == nil {
		return nil, ErrUnsupported
	}
	pool, ok := db.Statement.ConnPool.(*sql.DB)
	if !ok || pool == nil {
		return nil, ErrUnsupported
	}
	return &MySQLBackend{db: pool}, nil
}

const createSchema = "CREATE TABLE IF NOT EXISTS dlocks (" +
	"`key` VARBINARY(191) NOT NULL, token BINARY(32) NOT NULL, " +
	"expired_at DATETIME(6) NOT NULL, UNIQUE KEY uidx_key (`key`), " +
	"KEY idx_expired_at (expired_at)) ENGINE=InnoDB"

// Initialize creates an absent table and validates existing storage. It never
// alters an existing table. Run before serving requests.
func (m *MySQLBackend) Initialize(ctx context.Context) error {
	if m == nil || m.db == nil || ctx == nil {
		return ErrInvalidInput
	}
	if _, err := m.db.ExecContext(ctx, createSchema); err != nil {
		return err
	}
	var engine string
	if err := m.db.QueryRowContext(ctx, "SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='dlocks'").Scan(&engine); err != nil {
		return err
	}
	if engine != "InnoDB" {
		return ErrSchema
	}
	rows, err := m.db.QueryContext(ctx, "SELECT COLUMN_NAME,COLUMN_TYPE,IS_NULLABLE,EXTRA FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='dlocks'")
	if err != nil {
		return err
	}
	expected := map[string]string{"key": "varbinary(191)", "token": "binary(32)", "expired_at": "datetime(6)"}
	count := 0
	for rows.Next() {
		var name, typ, nullable, extra string
		if err = rows.Scan(&name, &typ, &nullable, &extra); err != nil {
			rows.Close()
			return err
		}
		if expected[name] != typ || nullable != "NO" || extra != "" {
			rows.Close()
			return ErrSchema
		}
		count++
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if count != 3 {
		return ErrSchema
	}
	rows, err = m.db.QueryContext(ctx, "SELECT INDEX_NAME,NON_UNIQUE,SEQ_IN_INDEX,COLUMN_NAME,COALESCE(SUB_PART,0) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='dlocks'")
	if err != nil {
		return err
	}
	defer rows.Close()
	type index struct {
		unique bool
		cols   []string
		prefix bool
	}
	indexes := map[string]*index{}
	for rows.Next() {
		var name, col string
		var nonunique, seq, prefix int
		if err = rows.Scan(&name, &nonunique, &seq, &col, &prefix); err != nil {
			return err
		}
		v := indexes[name]
		if v == nil {
			v = &index{unique: nonunique == 0}
			indexes[name] = v
		}
		v.cols = append(v.cols, col)
		v.prefix = v.prefix || prefix != 0
	}
	if err = rows.Err(); err != nil {
		return err
	}
	keyOK, expiryOK := false, false
	for _, v := range indexes {
		if v.unique {
			if len(v.cols) != 1 || v.cols[0] != "key" || v.prefix {
				return ErrSchema
			}
			keyOK = true
		}
		if len(v.cols) == 1 && v.cols[0] == "expired_at" && !v.prefix {
			expiryOK = true
		}
	}
	if !keyOK || !expiryOK {
		return ErrSchema
	}
	return nil
}
func (m *MySQLBackend) Acquire(ctx context.Context, key string, ttl time.Duration) (Grant, error) {
	if err := validate(ctx, key, ttl); err != nil {
		return Grant{}, err
	}
	if m == nil || m.db == nil {
		return Grant{}, ErrInvalidInput
	}
	token, err := newToken()
	if err != nil {
		return Grant{}, err
	}
	start := time.Now()
	result, err := m.db.ExecContext(ctx, "UPDATE dlocks SET token=?, expired_at=UTC_TIMESTAMP(6)+INTERVAL ? MICROSECOND WHERE `key`=? AND expired_at<=UTC_TIMESTAMP(6)", token[:], ttl.Microseconds(), []byte(key))
	if err != nil {
		return Grant{}, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return Grant{}, err
	}
	if n == 0 {
		_, err = m.db.ExecContext(ctx, "INSERT INTO dlocks (`key`,token,expired_at) VALUES (?,?,UTC_TIMESTAMP(6)+INTERVAL ? MICROSECOND)", []byte(key), token[:], ttl.Microseconds())
		if err != nil {
			var e *driver.MySQLError
			if errors.As(err, &e) && e.Number == 1062 {
				return Grant{}, ErrLockAcquisitionFailed
			}
			return Grant{}, err
		}
	}
	return finish(Grant{Key: key, Token: token}, start, ttl)
}
func (m *MySQLBackend) Renew(ctx context.Context, g Grant, ttl time.Duration) (Grant, error) {
	if err := validate(ctx, g.Key, ttl); err != nil {
		return Grant{}, err
	}
	if m == nil || m.db == nil || g.Token == ([32]byte{}) {
		return Grant{}, ErrInvalidInput
	}
	start := time.Now()
	if !start.Before(g.ValidUntil) {
		return Grant{}, ErrLeaseExpired
	}
	r, err := m.db.ExecContext(ctx, "UPDATE dlocks SET expired_at=UTC_TIMESTAMP(6)+INTERVAL ? MICROSECOND WHERE `key`=? AND token=? AND expired_at>UTC_TIMESTAMP(6)", ttl.Microseconds(), []byte(g.Key), g.Token[:])
	if err != nil {
		return Grant{}, err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return Grant{}, err
	}
	if n == 0 {
		return Grant{}, ErrLockNotHeld
	}
	if !time.Now().Before(g.ValidUntil) {
		return Grant{}, ErrLeaseExpired
	}
	return finish(g, start, ttl)
}
func finish(g Grant, start time.Time, ttl time.Duration) (Grant, error) {
	g.ValidUntil = start.Add(ttl)
	if !time.Now().Before(g.ValidUntil) {
		return Grant{}, ErrLeaseExpired
	}
	return g, nil
}
func (m *MySQLBackend) Release(ctx context.Context, g Grant) error {
	if err := validate(ctx, g.Key, 2*time.Microsecond); err != nil {
		return err
	}
	if m == nil || m.db == nil || g.Token == ([32]byte{}) {
		return ErrInvalidInput
	}
	r, err := m.db.ExecContext(ctx, "DELETE FROM dlocks WHERE `key`=? AND token=?", []byte(g.Key), g.Token[:])
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrLockNotHeld
	}
	return nil
}

var _ Backend = (*MySQLBackend)(nil)
