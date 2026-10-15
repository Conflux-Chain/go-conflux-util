package dlock

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	mysql "github.com/go-sql-driver/mysql"
)

// A transport-level driver lets us model a committed write with a lost
// acknowledgement without SQL string mocks or a new dependency.
type transport struct {
	calls []string
	exec  func(context.Context) (driver.Result, error)
}
type connector struct{ t *transport }

func (c connector) Connect(context.Context) (driver.Conn, error) { return transportConn{c.t}, nil }
func (c connector) Driver() driver.Driver                        { return transportDriver{c.t} }

type transportDriver struct{ t *transport }

func (d transportDriver) Open(string) (driver.Conn, error) { return transportConn{d.t}, nil }

type transportConn struct{ t *transport }

func (c transportConn) Prepare(string) (driver.Stmt, error) { return nil, ErrUnsupported }
func (c transportConn) Close() error                        { return nil }
func (c transportConn) Begin() (driver.Tx, error)           { return nil, ErrUnsupported }
func (c transportConn) ExecContext(ctx context.Context, q string, _ []driver.NamedValue) (driver.Result, error) {
	c.t.calls = append(c.t.calls, q)
	return c.t.exec(ctx)
}
func transportBackend(t *testing.T, tr *transport) *MySQLBackend {
	t.Helper()
	pool := sql.OpenDB(connector{tr})
	t.Cleanup(func() { pool.Close() })
	be := &MySQLBackend{db: pool}
	be.ready.Store(true)
	return be
}

type failedRandom struct{}

func (failedRandom) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func TestRandomFailureBeforeSQL(t *testing.T) {
	tr := &transport{exec: func(context.Context) (driver.Result, error) { t.Fatal("random failure reached SQL"); return nil, nil }}
	be := transportBackend(t, tr)
	be.random = failedRandom{}
	if g, err := be.Acquire(context.Background(), "key", time.Second); !errors.Is(err, io.ErrUnexpectedEOF) || g != (Grant{}) {
		t.Fatal(g, err)
	}
	if len(tr.calls) != 0 {
		t.Fatal("SQL executed")
	}
}
func TestUncertainAcknowledgementFailsClosed(t *testing.T) {
	tr := &transport{exec: func(context.Context) (driver.Result, error) { return driver.RowsAffected(1), nil }}
	be := transportBackend(t, tr)
	m, _ := NewLockManager(be)
	lease, err := m.Acquire(context.Background(), "key", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	tr.exec = func(context.Context) (driver.Result, error) { return nil, io.ErrUnexpectedEOF }
	if err = lease.Renew(context.Background(), time.Second); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatal(err)
	}
	before := len(tr.calls)
	if err = lease.Release(context.Background()); err != ErrLockNotHeld {
		t.Fatal(err)
	}
	if len(tr.calls) != before || lease.Remaining() != 0 {
		t.Fatal("uncertain renewal remained usable or issued compensation")
	}
	for _, q := range tr.calls {
		if strings.HasPrefix(q, "DELETE") {
			t.Fatal("compensating delete issued")
		}
	}
	if l, err := m.Acquire(context.Background(), "other", time.Second); l != nil || !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatal(l, err)
	}
}
func TestCancellationAfterWriteAndLateAck(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	tr := &transport{exec: func(context.Context) (driver.Result, error) { cancel(); return driver.RowsAffected(1), nil }}
	be := transportBackend(t, tr)
	if g, err := be.Acquire(ctx, "key", time.Second); g != (Grant{}) || !errors.Is(err, context.Canceled) {
		t.Fatal(g, err)
	}
	tr.exec = func(context.Context) (driver.Result, error) {
		time.Sleep(15 * time.Millisecond)
		return driver.RowsAffected(1), nil
	}
	if g, err := be.Acquire(context.Background(), "other", 5*time.Millisecond); g != (Grant{}) || err != ErrLeaseExpired {
		t.Fatal(g, err)
	}
}
func TestOnlyContentionNormalized(t *testing.T) {
	for _, number := range []uint16{1062, 1213, 1205, 1146} {
		n := 0
		dbErr := &mysql.MySQLError{Number: number, Message: "test failure"}
		tr := &transport{exec: func(context.Context) (driver.Result, error) {
			n++
			if n == 1 {
				return driver.RowsAffected(0), nil
			}
			return nil, dbErr
		}}
		be := transportBackend(t, tr)
		_, err := be.Acquire(context.Background(), "key", time.Second)
		if number == 1062 {
			if err != ErrLockAcquisitionFailed {
				t.Fatal(err)
			}
		} else if !errors.Is(err, dbErr) {
			t.Fatal("unexpected SQL error hidden", err)
		}
	}
}
