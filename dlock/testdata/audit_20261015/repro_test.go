package audit_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/dlock"
	driver "github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var ctx = context.Background()

func openDB(t *testing.T, params string) *gorm.DB {
	t.Helper()
	socket := os.Getenv("DLOCK_AUDIT_SOCKET")
	if socket == "" {
		t.Fatal("DLOCK_AUDIT_SOCKET must name an isolated audit server")
	}
	db, err := gorm.Open(mysql.Open("root@unix("+socket+")/audit_dlock?parseTime=true"+params), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&dlock.Dlock{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func mustAcquire(t *testing.T, be *dlock.MySQLBackend, key, nonce string, lease time.Duration) {
	t.Helper()
	ok, err := be.WriteEntry(ctx, key, nonce, lease)
	if err != nil || !ok {
		t.Fatalf("acquire(%q, %q): ok=%v err=%v", key, nonce, ok, err)
	}
}

func mustExec(t *testing.T, db *gorm.DB, sql string, args ...any) {
	t.Helper()
	if err := db.Session(&gorm.Session{NewDB: true}).Exec(sql, args...).Error; err != nil {
		t.Fatal(err)
	}
}

func TestAuditDifferentNoncesCanReleaseOrTakeOver(t *testing.T) {
	db := openDB(t, "")
	be := dlock.NewMySQLBackend(db)
	for i, other := range []string{"ownera", "OwnerA ", "ÓwnerA"} {
		key := fmt.Sprintf("collation-%d-%d", time.Now().UnixNano(), i)
		mustAcquire(t, be, key, "OwnerA", time.Minute)
		ok, err := be.DelEntry(ctx, key, other)
		if err != nil || !ok {
			t.Fatalf("expected reproduction: release with %q: ok=%v err=%v", other, ok, err)
		}
		t.Logf("wrong nonce %q deleted lock held with OwnerA", other)
		mustAcquire(t, be, key, "OwnerA", time.Minute)
		mustAcquire(t, be, key, other, time.Minute)
		t.Logf("wrong nonce %q also acquired unexpired lock", other)
	}
}

func TestAuditPositiveSubsecondLeaseExpiresBeforeReturn(t *testing.T) {
	db := openDB(t, "")
	err := db.Connection(func(conn *gorm.DB) error {
		// Pin the server session clock to a reproducible point within a second.
		mustExec(t, conn, "SET timestamp = 2000000000.950000")
		defer conn.Session(&gorm.Session{NewDB: true}).Exec("SET timestamp = 0")
		be := dlock.NewMySQLBackend(conn.Session(&gorm.Session{NewDB: true}))
		key := fmt.Sprintf("fraction-%d", time.Now().UnixNano())
		mustAcquire(t, be, key, "a", 100*time.Millisecond)
		var remaining int64
		if err := conn.Session(&gorm.Session{NewDB: true}).Raw("SELECT TIMESTAMPDIFF(MICROSECOND, NOW(6), expired_at) FROM dlocks WHERE `key` = ?", key).Scan(&remaining).Error; err != nil {
			return err
		}
		t.Logf("100 ms lease, remaining after successful Acquire = %d microseconds", remaining)
		if remaining >= 0 {
			t.Fatal("expected lease to already be expired")
		}
		// Advance only 50 ms. A second distinct owner can acquire within A's 100 ms requested lease.
		mustExec(t, conn, "SET timestamp = 2000000001.000000")
		ok, err := be.WriteEntry(ctx, key, "b", time.Second)
		t.Logf("second acquire after 50 ms: ok=%v err=%v", ok, err)
		if !ok || err != nil {
			t.Fatalf("expected B to acquire within A's requested lease: %v %v", ok, err)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAuditNonpositiveLeaseAccepted(t *testing.T) {
	db := openDB(t, "")
	be := dlock.NewMySQLBackend(db)
	for i, lease := range []time.Duration{0, -time.Second} {
		key := fmt.Sprintf("invalid-%d-%d", time.Now().UnixNano(), i)
		mustAcquire(t, be, key, "a", lease)
		if lease < 0 {
			mustAcquire(t, be, key, "b", time.Minute)
		}
		t.Logf("lease %v accepted as successfully acquired", lease)
	}
}

func TestAuditRowWaitReturnsExpiredLock(t *testing.T) {
	db := openDB(t, "")
	key := fmt.Sprintf("rowwait-%d", time.Now().UnixNano())
	be := dlock.NewMySQLBackend(db)
	mustAcquire(t, be, key, "old", time.Minute)
	mustExec(t, db, "UPDATE dlocks SET expired_at = NOW() - INTERVAL 1 MINUTE WHERE `key` = ?", key)
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	var row dlock.Dlock
	if err := tx.Raw("SELECT * FROM dlocks WHERE `key` = ? FOR UPDATE", key).Scan(&row).Error; err != nil {
		t.Fatal(err)
	}
	type result struct {
		ok      bool
		err     error
		elapsed time.Duration
	}
	ch := make(chan result, 1)
	go func() {
		start := time.Now()
		c, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		ok, err := be.WriteEntry(c, key, "a", time.Second)
		ch <- result{ok, err, time.Since(start)}
	}()
	// Wait until MySQL proves the second transaction is blocked on our row.
	deadline := time.Now().Add(5 * time.Second)
	for {
		var n int64
		if err := db.Raw("SELECT COUNT(*) FROM performance_schema.data_lock_waits").Scan(&n).Error; err != nil {
			t.Fatal(err)
		}
		if n > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("did not observe row lock wait")
		}
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(2200 * time.Millisecond)
	if err := tx.Commit().Error; err != nil {
		t.Fatal(err)
	}
	r := <-ch
	if !r.ok || r.err != nil {
		t.Fatalf("acquire result: %+v", r)
	}
	var remaining int64
	if err := db.Raw("SELECT TIMESTAMPDIFF(MICROSECOND, NOW(6), expired_at) FROM dlocks WHERE `key` = ?", key).Scan(&remaining).Error; err != nil {
		t.Fatal(err)
	}
	if remaining >= 0 {
		t.Fatalf("unexpected live lease: %d", remaining)
	}
	mustAcquire(t, be, key, "b", time.Minute)
	t.Logf("A returned success after %v with remaining=%d us; B immediately acquired successfully", r.elapsed, remaining)
}

func TestAuditConcurrentFirstInsertIsDatabaseError(t *testing.T) {
	db := openDB(t, "")
	key := fmt.Sprintf("insert-race-%d", time.Now().UnixNano())
	// Pause both real SELECTs after execution, before the real INSERTs.
	var barrier sync.WaitGroup
	barrier.Add(2)
	if err := db.Callback().Query().After("gorm:query").Register("audit:barrier", func(tx *gorm.DB) {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			barrier.Done()
			barrier.Wait()
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer db.Callback().Query().Remove("audit:barrier")
	lm := dlock.NewLockManager(dlock.NewMySQLBackend(db))
	errs := make(chan error, 2)
	for _, nonce := range []string{"a", "b"} {
		go func(nonce string) { errs <- lm.Acquire(ctx, dlock.NewLockIntent(key, nonce, time.Minute)) }(nonce)
	}
	first, second := <-errs, <-errs
	if first == nil {
		first = second
	} else if second != nil {
		t.Fatalf("both failed: %v / %v", first, second)
	}
	var mysqlErr *driver.MySQLError
	if !errors.As(first, &mysqlErr) || mysqlErr.Number != 1062 {
		t.Fatalf("expected duplicate-key error, got %v", first)
	}
	if errors.Is(first, dlock.ErrLockAcquisitionFailed) {
		t.Fatal("unexpected correct error mapping")
	}
	t.Logf("loser received %v; errors.Is(ErrLockAcquisitionFailed)=false", first)
}

func TestAuditStaleReleaseDeletesNewGeneration(t *testing.T) {
	db := openDB(t, "")
	be := dlock.NewMySQLBackend(db)
	key := fmt.Sprintf("aba-%d", time.Now().UnixNano())
	mustAcquire(t, be, key, "node0", time.Minute)
	oldIntent := dlock.NewLockIntent(key, "node0", time.Minute)
	ok, err := be.DelEntry(ctx, key, "node0")
	if !ok || err != nil {
		t.Fatal(ok, err)
	}
	mustAcquire(t, be, key, "node0", time.Minute)
	err = dlock.NewLockManager(be).Release(ctx, oldIntent)
	if err != nil {
		t.Fatal(err)
	}
	mustAcquire(t, be, key, "node1", time.Minute)
	t.Log("delayed release from generation 1 deleted generation 2; node1 acquired")
}

func TestAuditSessionTimeZonesBreakExclusion(t *testing.T) {
	utc := openDB(t, "&time_zone=%27%2B00%3A00%27")
	plus8 := openDB(t, "&time_zone=%27%2B08%3A00%27")
	key := fmt.Sprintf("timezone-%d", time.Now().UnixNano())
	mustAcquire(t, dlock.NewMySQLBackend(utc), key, "a", time.Minute)
	mustAcquire(t, dlock.NewMySQLBackend(plus8), key, "b", time.Minute)
	t.Log("UTC client A and UTC+8 client B both acquired a 1 minute lease without waiting")
}

func TestAuditFactoryAcceptsSQLite(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("store.sqlite.path", t.TempDir()+"/locks.db")
	lm := dlock.NewLockManagerFromViper()
	err := lm.Acquire(ctx, dlock.NewLockIntent("factory", "owner", time.Second))
	if err == nil || !strings.Contains(err.Error(), "syntax error") {
		t.Fatalf("unexpected result: %v", err)
	}
	t.Logf("factory succeeded but Acquire failed: %v", err)
}

func TestAuditVersionResetsAfterRelease(t *testing.T) {
	db := openDB(t, "")
	be := dlock.NewMySQLBackend(db)
	key := fmt.Sprintf("version-%d", time.Now().UnixNano())
	mustAcquire(t, be, key, "a", time.Minute)
	mustAcquire(t, be, key, "a", time.Minute)
	var before, after dlock.Dlock
	if err := db.Where("`key` = ?", key).First(&before).Error; err != nil {
		t.Fatal(err)
	}
	if ok, err := be.DelEntry(ctx, key, "a"); !ok || err != nil {
		t.Fatal(ok, err)
	}
	mustAcquire(t, be, key, "b", time.Minute)
	if err := db.Where("`key` = ?", key).First(&after).Error; err != nil {
		t.Fatal(err)
	}
	if before.Version != 1 || after.Version != 0 {
		t.Fatalf("versions: %d -> %d", before.Version, after.Version)
	}
	t.Logf("Version decreased across ownership generations: %d -> %d", before.Version, after.Version)
}

func TestAuditControls(t *testing.T) {
	db := openDB(t, "")
	be := dlock.NewMySQLBackend(db)
	key := fmt.Sprintf("controls-%d", time.Now().UnixNano())
	mustAcquire(t, be, key, "a", time.Minute)
	if ok, err := be.WriteEntry(ctx, key, "b", time.Minute); ok || err != nil {
		t.Fatal(ok, err)
	}
	if ok, err := be.DelEntry(ctx, key, "b"); ok || err != nil {
		t.Fatal(ok, err)
	}
	mustExec(t, db, "UPDATE dlocks SET expired_at = NOW() - INTERVAL 1 MINUTE WHERE `key` = ?", key)
	var wg sync.WaitGroup
	results := make(chan bool, 16)
	errorsCh := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ok, err := be.WriteEntry(ctx, key, fmt.Sprintf("contender-%d", i), time.Minute)
			results <- ok
			errorsCh <- err
		}(i)
	}
	wg.Wait()
	winners := 0
	for i := 0; i < 16; i++ {
		if <-results {
			winners++
		}
		if err := <-errorsCh; err != nil {
			t.Fatal(err)
		}
	}
	if winners != 1 {
		t.Fatalf("winners=%d", winners)
	}
	if ok, err := be.DelEntry(ctx, key, "a"); ok || err != nil {
		t.Fatal(ok, err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if ok, err := be.WriteEntry(cancelled, key+"-cancelled", "a", time.Minute); ok || !errors.Is(err, context.Canceled) {
		t.Fatal(ok, err)
	}
	t.Log("distinct nonce exclusion, wrong/stale nonce release rejection, 16-way expired takeover, and context cancellation controls passed")
}
