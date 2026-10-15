package dlock

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/store"
	"github.com/Conflux-Chain/go-conflux-util/viper"
	driver "github.com/go-sql-driver/mysql"
)

// NewLockManagerFromViper returns configuration/connection/schema errors to the
// caller. The manager owns the resulting pool. No implicit SQLite fallback.
func NewLockManagerFromViper() (*LockManager, error) {
	var conf store.Config
	if err := viper.UnmarshalKey("store", &conf); err != nil {
		return nil, err
	}
	return NewLockManagerFromConfig(context.Background(), conf)
}

// NewLockManagerFromConfig opens an owned pool and initializes the schema.
// Explicit config uses pool settings as supplied; Viper applies its defaults.
func NewLockManagerFromConfig(ctx context.Context, conf store.Config) (*LockManager, error) {
	if ctx == nil || conf.Mysql == nil || conf.Sqlite != nil || conf.Mysql.Database == "" {
		return nil, ErrUnsupported
	}
	c := driver.NewConfig()
	c.User = conf.Mysql.Username
	c.Passwd = conf.Mysql.Password
	c.Net = "tcp"
	c.Addr = conf.Mysql.Host
	c.ParseTime = true
	c.Loc = time.UTC
	if conf.Mysql.Location != "" {
		loc, err := time.LoadLocation(conf.Mysql.Location)
		if err != nil {
			return nil, err
		}
		c.Loc = loc
	}
	if conf.Mysql.AutoCreate {
		pool, err := sql.Open("mysql", c.FormatDSN())
		if err != nil {
			return nil, err
		}
		// Identifiers cannot be parameters; escape the trusted configuration name.
		_, err = pool.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS `"+strings.ReplaceAll(conf.Mysql.Database, "`", "``")+"`")
		closeErr := pool.Close()
		if err != nil {
			return nil, fmt.Errorf("create lock database: %w", err)
		}
		if closeErr != nil {
			return nil, closeErr
		}
	}
	c.DBName = conf.Mysql.Database
	pool, err := sql.Open("mysql", c.FormatDSN())
	if err != nil {
		return nil, err
	}
	pool.SetMaxOpenConns(conf.MaxOpenConns)
	pool.SetMaxIdleConns(conf.MaxIdleConns)
	pool.SetConnMaxLifetime(conf.ConnMaxLifetime)
	return ownPool(ctx, pool)
}
func ownPool(ctx context.Context, pool *sql.DB) (*LockManager, error) {
	be := &MySQLBackend{db: pool}
	if err := be.Initialize(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	m, err := NewLockManager(be)
	if err != nil {
		pool.Close()
		return nil, err
	}
	m.closeFn = pool.Close
	return m, nil
}

// Close waits for in-flight operations, rejects subsequent requests, and closes
// only factory-owned pools. Stop business tasks and release leases first.
func (m *LockManager) Close() error {
	if m == nil {
		return ErrInvalidInput
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.closed = true
		if m.closeFn != nil {
			m.closeErr = m.closeFn()
		}
	}
	return m.closeErr
}
