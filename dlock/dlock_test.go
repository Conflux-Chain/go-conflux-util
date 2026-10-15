//go:build integration

package dlock

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"os"
	"testing"
	"time"

	driver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var testMysqlBe *MySQLBackend

func TestMain(m *testing.M) {
	if os.Getenv("TEST_MYSQL_HOST") == "" || os.Getenv("TEST_MYSQL_DB") == "" {
		panic("integration requires TEST_MYSQL_HOST and TEST_MYSQL_DB (dedicated test database)")
	}
	cfg := driver.NewConfig()
	cfg.User = os.Getenv("TEST_MYSQL_USER")
	cfg.Passwd = os.Getenv("TEST_MYSQL_PWD")
	cfg.Net = "tcp"
	if os.Getenv("TEST_MYSQL_NET") != "" {
		cfg.Net = os.Getenv("TEST_MYSQL_NET")
	}
	cfg.Addr = os.Getenv("TEST_MYSQL_HOST")
	cfg.DBName = os.Getenv("TEST_MYSQL_DB")
	cfg.ParseTime = true
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	testMysqlBe, err = NewMySQLBackend(db)
	if err != nil {
		panic(err)
	}
	if err = testMysqlBe.Initialize(context.Background()); err != nil {
		panic(err)
	}
	code := m.Run()
	testMysqlBe.db.Close()
	os.Exit(code)
}
func uniqueKey(t testing.TB) string {
	t.Helper()
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	key := hex.EncodeToString(b[:])
	t.Cleanup(func() {
		if _, err := testMysqlBe.db.Exec("DELETE FROM dlocks WHERE `key` LIKE ?", []byte(key+"%")); err != nil {
			t.Error(err)
		}
	})
	return key
}
func TestMysqlGenerationAndBinaryIdentity(t *testing.T) {
	ctx := context.Background()
	key := uniqueKey(t)
	grants := []Grant{}
	for _, suffix := range []string{"A", "a", "A ", "Á"} {
		g, err := testMysqlBe.Acquire(ctx, key+suffix, time.Second)
		if err != nil {
			t.Fatal(err)
		}
		grants = append(grants, g)
	}
	if _, err := testMysqlBe.Acquire(ctx, key+"A", time.Second); err != ErrLockAcquisitionFailed {
		t.Fatal(err)
	}
	old := grants[0]
	if err := testMysqlBe.Release(ctx, old); err != nil {
		t.Fatal(err)
	}
	fresh, err := testMysqlBe.Acquire(ctx, old.Key, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Token == old.Token {
		t.Fatal("token reused")
	}
	if err = testMysqlBe.Release(ctx, old); err != ErrLockNotHeld {
		t.Fatal("old generation deleted new", err)
	}
	if _, err = testMysqlBe.Renew(ctx, old, time.Second); err != ErrLockNotHeld {
		t.Fatal("old generation renewed new", err)
	}
}
func TestRejectTransaction(t *testing.T) {
	cfg := &gorm.Config{}
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: testMysqlBe.db, SkipInitializeWithVersion: true}), cfg)
	if err != nil {
		t.Fatal(err)
	}
	tx := db.Begin()
	defer tx.Rollback()
	if _, err = NewMySQLBackend(tx); err != ErrUnsupported {
		t.Fatal(err)
	}
}

func TestFractionalLeases(t *testing.T) {
	ctx := context.Background()
	for _, ttl := range []time.Duration{100 * time.Millisecond, 500 * time.Millisecond, time.Second} {
		key := uniqueKey(t)
		g, err := testMysqlBe.Acquire(ctx, key, ttl)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = testMysqlBe.Acquire(ctx, key, ttl); err != ErrLockAcquisitionFailed {
			t.Fatal("early takeover", err)
		}
		time.Sleep(ttl + 5*time.Millisecond)
		if _, err = testMysqlBe.Renew(ctx, g, time.Second); err != ErrLeaseExpired {
			t.Fatal("resurrected expired grant", err)
		}
		if _, err = testMysqlBe.Acquire(ctx, key, time.Second); err != nil {
			t.Fatal("failed expired takeover", err)
		}
	}
}
func TestWaitConsumesAcquireBudget(t *testing.T) {
	ctx := context.Background()
	key := uniqueKey(t)
	g, err := testMysqlBe.Acquire(ctx, key, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = testMysqlBe.db.Exec("UPDATE dlocks SET expired_at=UTC_TIMESTAMP(6)-INTERVAL 1 SECOND WHERE `key`=?", []byte(key))
	if err != nil {
		t.Fatal(err)
	}
	tx, err := testMysqlBe.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	var token []byte
	if err = tx.QueryRow("SELECT token FROM dlocks WHERE `key`=? FOR UPDATE", []byte(key)).Scan(&token); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, e := testMysqlBe.Acquire(ctx, g.Key, 50*time.Millisecond); done <- e }()
	time.Sleep(120 * time.Millisecond)
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err = <-done; err != ErrLeaseExpired {
		t.Fatal("wait returned usable expired lease", err)
	}
}

func TestConcurrentAcquire(t *testing.T) {
	ctx := context.Background()
	for _, expired := range []bool{false, true} {
		key := uniqueKey(t)
		if expired {
			if _, err := testMysqlBe.Acquire(ctx, key, time.Second); err != nil {
				t.Fatal(err)
			}
			if _, err := testMysqlBe.db.Exec("UPDATE dlocks SET expired_at=UTC_TIMESTAMP(6)-INTERVAL 1 SECOND WHERE `key`=?", []byte(key)); err != nil {
				t.Fatal(err)
			}
		}
		results := make(chan error, 16)
		start := make(chan struct{})
		for i := 0; i < 16; i++ {
			go func() { <-start; _, err := testMysqlBe.Acquire(ctx, key, time.Second); results <- err }()
		}
		close(start)
		wins := 0
		for i := 0; i < 16; i++ {
			err := <-results
			if err == nil {
				wins++
			} else if err != ErrLockAcquisitionFailed {
				t.Fatal("infrastructure error must not be hidden", err)
			}
		}
		if wins != 1 {
			t.Fatalf("expected one winner, got %d", wins)
		}
	}
}
func TestDirectBackendValidation(t *testing.T) {
	for _, ttl := range []time.Duration{0, -1, time.Microsecond, time.Second + 1} {
		if _, err := testMysqlBe.Acquire(context.Background(), uniqueKey(t), ttl); err != ErrInvalidInput {
			t.Fatal(err)
		}
		if _, err := testMysqlBe.Renew(context.Background(), Grant{Key: "k", Token: [32]byte{1}}, ttl); err != ErrInvalidInput {
			t.Fatal(err)
		}
	}
	key := uniqueKey(t)
	if _, err := testMysqlBe.Acquire(context.Background(), key+string(make([]byte, 192)), time.Second); err != ErrInvalidInput {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := testMysqlBe.Acquire(ctx, key, time.Second); err != context.Canceled {
		t.Fatal(err)
	}
	closed, err := sql.Open("mysql", "")
	if err != nil {
		t.Fatal(err)
	}
	closed.Close()
	if _, err = (&MySQLBackend{db: closed}).Acquire(context.Background(), key, time.Second); err == nil || err == ErrLockAcquisitionFailed {
		t.Fatal("database error normalized", err)
	}
}
