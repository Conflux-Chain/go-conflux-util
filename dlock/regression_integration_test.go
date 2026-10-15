//go:build integration

package dlock

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	driver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func isolatedBackend(t *testing.T) *MySQLBackend {
	t.Helper()
	name := "dlock_v2_" + uniqueKey(t)
	if _, err := testMysqlBe.db.Exec("CREATE DATABASE " + name); err != nil {
		t.Fatal(err)
	}
	cfg, err := driver.ParseDSN(testDSN)
	if err != nil {
		t.Fatal(err)
	}
	cfg.DBName = name
	pool, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		if _, err := testMysqlBe.db.Exec("DROP DATABASE " + name); err != nil {
			t.Error(err)
		}
	})
	return &MySQLBackend{db: pool}
}
func execTest(t *testing.T, db *sql.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatal(err)
	}
}
func TestSchemaDefinitionsAndNoMutation(t *testing.T) {
	cases := []struct {
		name, ddl string
		valid     bool
	}{
		{"renamed_indexes", strings.ReplaceAll(strings.ReplaceAll(createSchema, "uidx_key", "legacy_unique"), "idx_expired_at", "legacy_expiry"), true},
		{"wrong_named_unique", strings.Replace(createSchema, "UNIQUE KEY uidx_key (`key`)", "UNIQUE KEY uidx_key (token)", 1), false},
		{"missing_unique", strings.Replace(createSchema, "UNIQUE KEY uidx_key", "KEY uidx_key", 1), false},
		{"prefix_unique", strings.Replace(createSchema, "uidx_key (`key`)", "uidx_key (`key`(10))", 1), false},
		{"composite_unique", strings.Replace(createSchema, "uidx_key (`key`)", "uidx_key (`key`,token)", 1), false},
		{"precision", strings.Replace(createSchema, "DATETIME(6)", "DATETIME(3)", 1), false},
		{"key_text", strings.Replace(createSchema, "VARBINARY(191)", "VARCHAR(191)", 1), false},
		{"token_width", strings.Replace(createSchema, "BINARY(32)", "BINARY(16)", 1), false},
		{"missing_expiry_index", strings.Replace(createSchema, ", KEY idx_expired_at (expired_at)", "", 1), false},
		{"invisible_expiry", strings.Replace(createSchema, "idx_expired_at (expired_at)", "idx_expired_at (expired_at) INVISIBLE", 1), false},
		{"wrong_engine", strings.Replace(createSchema, "InnoDB", "MyISAM", 1), false},
		{"old_schema", "CREATE TABLE dlocks (id BIGINT, `key` VARCHAR(191) NOT NULL, nonce VARCHAR(191), expired_at DATETIME, UNIQUE KEY old_key (`key`)) ENGINE=InnoDB", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			be := isolatedBackend(t)
			execTest(t, be.db, c.ddl)
			var table, before, after string
			if err := be.db.QueryRow("SHOW CREATE TABLE dlocks").Scan(&table, &before); err != nil {
				t.Fatal(err)
			}
			err := be.Initialize(context.Background())
			if c.valid && err != nil {
				t.Fatal(err)
			}
			if !c.valid && !errors.Is(err, ErrSchema) {
				t.Fatal("bad schema accepted", err)
			}
			if err = be.db.QueryRow("SHOW CREATE TABLE dlocks").Scan(&table, &after); err != nil {
				t.Fatal(err)
			}
			if before != after {
				t.Fatal("initializer mutated existing constraints")
			}
		})
	}
}
func TestConcurrentInitializeAndMigrationRecovery(t *testing.T) {
	ctx := context.Background()
	be := isolatedBackend(t)
	results := make(chan error, 8)
	for i := 0; i < 8; i++ {
		go func() { results <- be.Initialize(ctx) }()
	}
	for i := 0; i < 8; i++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	execTest(t, be.db, "DROP TABLE dlocks")
	execTest(t, be.db, "CREATE TABLE dlocks (`key` VARCHAR(191),nonce VARCHAR(191),UNIQUE KEY old_key (`key`))")
	execTest(t, be.db, "INSERT INTO dlocks VALUES ('old','secret')")
	// Exercise the checked-in offline SQL, not an independent migration copy.
	content, err := os.ReadFile("migrations/v2.sql")
	if err != nil {
		t.Fatal(err)
	}
	lines := []string{}
	for _, line := range strings.Split(string(content), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "--") {
			lines = append(lines, line)
		}
	}
	stmts := strings.Split(strings.Join(lines, "\n"), ";")
	execTest(t, be.db, stmts[0]) // interruption after RENAME
	if err = be.Initialize(ctx); err != nil {
		t.Fatal("recovery after RENAME", err)
	}
	execTest(t, be.db, strings.Replace(stmts[1], "CREATE TABLE dlocks", "CREATE TABLE IF NOT EXISTS dlocks", 1))
	var oldRows, newRows int
	if err = be.db.QueryRow("SELECT COUNT(*) FROM dlocks_v1_backup").Scan(&oldRows); err != nil {
		t.Fatal(err)
	}
	if err = be.db.QueryRow("SELECT COUNT(*) FROM dlocks").Scan(&newRows); err != nil {
		t.Fatal(err)
	}
	if oldRows != 1 || newRows != 0 {
		t.Fatal("migration copied leases or lost backup")
	}
	execTest(t, be.db, "RENAME TABLE dlocks TO dlocks_v2_backup, dlocks_v1_backup TO dlocks")
	execTest(t, be.db, "TRUNCATE TABLE dlocks")
	if err = be.Initialize(ctx); !errors.Is(err, ErrSchema) {
		t.Fatal("v2 accepted rollback schema", err)
	}
}
func TestControlledMicrosecondBoundary(t *testing.T) {
	be := isolatedBackend(t)
	be.db.SetMaxOpenConns(1)
	if err := be.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	execTest(t, be.db, "SET timestamp=2000000000.950000")
	t.Cleanup(func() { execTest(t, be.db, "SET timestamp=0") })
	g, err := be.Acquire(context.Background(), "boundary", 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	execTest(t, be.db, "SET timestamp=2000000001.000000")
	if _, err = be.Acquire(context.Background(), g.Key, time.Second); err != ErrLockAcquisitionFailed {
		t.Fatal("truncated positive TTL", err)
	}
	execTest(t, be.db, "SET timestamp=2000000001.050000")
	if _, err = be.Renew(context.Background(), g, time.Second); err != ErrLockNotHeld {
		t.Fatal("renew at expiration succeeded", err)
	}
	if _, err = be.Acquire(context.Background(), g.Key, time.Second); err != nil {
		t.Fatal("exact boundary not expired", err)
	}
}
func sessionBackend(t *testing.T, zone string, foundRows bool) *MySQLBackend {
	t.Helper()
	cfg, err := driver.ParseDSN(testDSN)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ClientFoundRows = foundRows
	pool, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	pool.SetMaxIdleConns(1)
	t.Cleanup(func() { pool.Close() })
	execTest(t, pool, "SET time_zone=?", zone)
	be := &MySQLBackend{db: pool}
	if err = be.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	return be
}
func TestTimezoneOffsetsAndClientFoundRows(t *testing.T) {
	// Fixed offsets also cover US winter/summer offsets without requiring
	// optional server timezone tables. Lease SQL always uses UTC.
	for _, zone := range []string{"+08:00", "-05:00", "-04:00"} {
		t.Run(zone, func(t *testing.T) {
			utc := sessionBackend(t, "+00:00", false)
			other := sessionBackend(t, zone, true)
			key := uniqueKey(t)
			g, err := utc.Acquire(context.Background(), key, time.Second)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = other.Acquire(context.Background(), key, time.Second); err != ErrLockAcquisitionFailed {
				t.Fatal("timezone allowed early takeover", err)
			}
			g, err = other.Renew(context.Background(), g, time.Second)
			if err != nil {
				t.Fatal(err)
			}
			if err = utc.Release(context.Background(), g); err != nil {
				t.Fatal(err)
			}
			fresh, err := other.Acquire(context.Background(), key, time.Second)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = utc.Renew(context.Background(), g, time.Second); err != ErrLockNotHeld {
				t.Fatal("old token renewed", err)
			}
			if err = utc.Release(context.Background(), fresh); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func waitForRowWait(t *testing.T, blocker int64) {
	t.Helper()
	until := time.Now().Add(5 * time.Second)
	for time.Now().Before(until) {
		var count int
		err := testMysqlBe.db.QueryRow("SELECT COUNT(*) FROM performance_schema.data_lock_waits w JOIN performance_schema.threads t ON t.THREAD_ID=w.BLOCKING_THREAD_ID WHERE t.PROCESSLIST_ID=?", blocker).Scan(&count)
		if err != nil {
			t.Fatal(err)
		}
		if count > 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("database never reported expected row wait")
}
func TestRenewWaitAndCancelledAcquisition(t *testing.T) {
	for _, renew := range []bool{false, true} {
		t.Run(map[bool]string{true: "renew", false: "cancel"}[renew], func(t *testing.T) {
			ctx := context.Background()
			key := uniqueKey(t)
			g, err := testMysqlBe.Acquire(ctx, key, 200*time.Millisecond)
			if err != nil {
				t.Fatal(err)
			}
			if !renew {
				execTest(t, testMysqlBe.db, "UPDATE dlocks SET expired_at=UTC_TIMESTAMP(6)-INTERVAL 1 SECOND WHERE `key`=?", []byte(key))
			}
			tx, err := testMysqlBe.db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			var blocker int64
			if err = tx.QueryRow("SELECT CONNECTION_ID()").Scan(&blocker); err != nil {
				t.Fatal(err)
			}
			var token []byte
			if err = tx.QueryRow("SELECT token FROM dlocks WHERE `key`=? FOR UPDATE", []byte(key)).Scan(&token); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			opCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			go func() {
				var e error
				if renew {
					_, e = testMysqlBe.Renew(opCtx, g, time.Second)
				} else {
					_, e = testMysqlBe.Acquire(opCtx, key, 200*time.Millisecond)
				}
				done <- e
			}()
			waitForRowWait(t, blocker)
			if renew {
				time.Sleep(220 * time.Millisecond)
				if err = tx.Commit(); err != nil {
					t.Fatal(err)
				}
				if err = <-done; err != ErrLeaseExpired {
					t.Fatal("renew crossed old deadline", err)
				}
			} else {
				cancel()
				if err = <-done; !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				if err = tx.Commit(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
func TestCleanupRechecksRenewedRow(t *testing.T) {
	be := isolatedBackend(t)
	ctx := context.Background()
	if err := be.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	g, err := be.Acquire(ctx, "cleanup", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	execTest(t, be.db, "UPDATE dlocks SET expired_at=UTC_TIMESTAMP(6)-INTERVAL 1 SECOND")
	tx, err := be.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	var blocker int64
	if err = tx.QueryRow("SELECT CONNECTION_ID()").Scan(&blocker); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec("UPDATE dlocks SET expired_at=UTC_TIMESTAMP(6)+INTERVAL 1 HOUR"); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		n, e := be.CleanupExpired(ctx, 1)
		if e == nil && n != 0 {
			e = errors.New("deleted freshly renewed row")
		}
		done <- e
	}()
	// InnoDB may skip the row using semi-consistent/index reads, or wait
	// and re-check it. Both must preserve the renewed record.
	completed := false
	select {
	case err = <-done:
		completed = true
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(20 * time.Millisecond):
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if !completed {
		select {
		case err = <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("cleanup did not finish")
		}
	}

	if err = be.Release(ctx, g); err != nil {
		t.Fatal("live record disappeared", err)
	}
}
func TestSharedLeaseConcurrentRenewReleaseAndRecovery(t *testing.T) {
	ctx := context.Background()
	m, _ := NewLockManager(testMysqlBe)
	key := uniqueKey(t)
	old, err := m.Acquire(ctx, key, 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(30 * time.Millisecond)
	fresh, err := m.Acquire(ctx, key, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err = old.Release(ctx); err != ErrLockNotHeld {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := fresh.Renew(ctx, time.Second)
			if err != nil && err != ErrLeaseExpired {
				t.Error(err)
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := fresh.Release(ctx); err != nil {
			t.Error(err)
		}
	}()
	wg.Wait()
	if _, err = m.Acquire(ctx, key, time.Second); err != nil {
		t.Fatal(err)
	}
}
func TestFixedConnectionRejectedAndOwnedPoolClosed(t *testing.T) {
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: testMysqlBe.db, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	err = db.Connection(func(conn *gorm.DB) error {
		_, e := NewMySQLBackend(conn)
		if e != ErrUnsupported {
			t.Fatal(e)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := sql.Open("mysql", testDSN)
	if err != nil {
		t.Fatal(err)
	}
	m, err := ownPool(context.Background(), pool)
	if err != nil {
		t.Fatal(err)
	}
	if err = m.Close(); err != nil {
		t.Fatal(err)
	}
	if err = m.Close(); err != nil {
		t.Fatal(err)
	}
	if err = pool.Ping(); err == nil {
		t.Fatal("owned pool remained open")
	}
}
func TestNonStrictByteLengthBoundary(t *testing.T) {
	be := sessionBackend(t, "+00:00", false)
	execTest(t, be.db, "SET sql_mode=''")
	key := uniqueKey(t) + strings.Repeat("é", 79) + "x" // 32 + 158 + 1 = 191 bytes
	g, err := be.Acquire(context.Background(), key, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = be.Acquire(context.Background(), key+" ", time.Second); err != ErrInvalidInput {
		t.Fatal("overlength not rejected", err)
	}
	if err = be.Release(context.Background(), g); err != nil {
		t.Fatal(err)
	}
}

func TestInitializationIsRequiredForEveryBackend(t *testing.T) {
	ctx := context.Background()
	be := isolatedBackend(t)
	g := Grant{Key: "uninitialized", Token: [32]byte{1}, ValidUntil: time.Now().Add(time.Second)}
	if _, err := be.Acquire(ctx, g.Key, time.Second); err != ErrSchema {
		t.Fatal("Acquire bypassed initialization", err)
	}
	if _, err := be.Renew(ctx, g, time.Second); err != ErrSchema {
		t.Fatal("Renew bypassed initialization", err)
	}
	if err := be.Release(ctx, g); err != ErrSchema {
		t.Fatal("Release bypassed initialization", err)
	}
	if _, err := be.CleanupExpired(ctx, 1); err != ErrSchema {
		t.Fatal("cleanup bypassed initialization", err)
	}
	if err := be.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := be.Acquire(ctx, g.Key, time.Second); err != nil {
		t.Fatal(err)
	}
	// Detecting a later incompatibility must revoke readiness too.
	execTest(t, be.db, "ALTER TABLE dlocks MODIFY `key` VARCHAR(191) NOT NULL")
	if err := be.Initialize(ctx); err != ErrSchema {
		t.Fatal(err)
	}
	if _, err := be.Acquire(ctx, g.Key, time.Second); err != ErrSchema {
		t.Fatal("failed revalidation did not revoke readiness", err)
	}
}
