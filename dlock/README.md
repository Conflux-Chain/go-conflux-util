# MySQL leases (v2, breaking API/schema change)

dlock coordinates expiring leases in one authoritative MySQL database. It does
**not** guarantee atomicity of business operations or strict mutual exclusion
after a pause. A previous holder can resume after expiration while another
holder runs. Cancellation is cooperative. Downstream resources must validate
their own durable, monotonically increasing fencing token if stale writes must
be rejected. The secret ownership token here is random, not a fencing token.

## Construction and use

The complete example in [example_test.go](example_test.go) is compiled by normal
Go tests. Configure Viper's `store.mysql` (host, username, password, database);
`NewLockManagerFromViper() (*LockManager, error)` creates and owns its pool.
Missing MySQL, SQLite configuration, decode/connection/schema errors are
returned, never Fatal. Explicit callers can use
`NewLockManagerFromConfig(ctx, store.Config{Mysql: ...})`; pool settings are used
as supplied. Set connect/operation deadlines appropriate to your deployment.

For an externally owned GORM pool:

```go
backend, err := dlock.NewMySQLBackend(db)
if err != nil { return err }
if err := backend.Initialize(ctx); err != nil { return err }
manager, err := dlock.NewLockManager(backend)
if err != nil { return err }
defer manager.Close() // does not close db
lease, err := manager.Acquire(ctx, "job", 5*time.Second)
if err != nil { return err } // never run work on acquisition failure
defer func() {
    releaseCtx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()
    if err := lease.Release(releaseCtx); err != nil { log.Print(err) }
}()
if err := lease.Renew(ctx, 5*time.Second); err != nil { return err }
// Perform bounded work, honoring lease.ValidUntil() and cancellation.
```

Only MySQL GORM handles whose Statement.ConnPool is an independent `*sql.DB`
are accepted. Transactions, fixed connections and prepared/routed pool wrappers
are rejected. Clauses, table scopes, callbacks, DryRun, loggers and GORM routing
do not affect lock SQL. Use a dedicated primary pool, never a read replica.

Every Acquire generates a fresh secret 32-byte token. The Lease hides it;
Renew/Release on one handle serialize. Do not copy a Lease. Release makes that
handle terminal, including on uncertain errors. Renewal failure makes the
handle terminal once a database operation was attempted. Validation failure
before SQL leaves its previous deadline unchanged, but callers should stop work
on any renewal error. No implicit same-nonce reacquisition/renewal remains.
Backend implementers receive secret Grant capabilities and must not log them
or reuse tokens. Do not expose a custom Backend's Grant to untrusted code.

## Time and failure contract

Keys are 1..191 **bytes**, compared byte-for-byte: case, accents and trailing
spaces identify different keys. TTL must be positive, an exact number of
microseconds, and greater than 1 microsecond (the minimum safety margin).
Invalid values are rejected before SQL at both manager and backend entrances.

SQL uses UTC_TIMESTAMP(6), DATETIME(6) and integer microsecond intervals;
expiration is `expired_at <= now`. Session timezone/DST does not change lease
identity or duration. Keep database clocks disciplined: arbitrary wall-clock
jumps, failover/replication history loss and split-brain are outside this lease
protocol's guarantee. A 1% margin does not compensate for arbitrary clock jumps.

The conservative local cutoff is operation start + TTL -
max(1 microsecond, TTL/100), using Go's monotonic clock. Requests which finish
after this cutoff fail; renewal must also finish before the prior cutoff.
ValidUntil/Remaining are advisory snapshots, not an assurance against a pause
immediately after a check. Re-check on resume, stop work on expiry or uncertainty,
and use downstream fencing when required. A returned success does not promise
unbounded time after return.

Timeout/cancellation/network errors can leave a committed but unacknowledged
lease. Do not continue work or retry using the old generation. There is no
compensating delete; abandoned leases recover by expiration. Contention returns
ErrLockAcquisitionFailed; deadlock, connection and other unexpected SQL errors
are preserved for errors.Is/errors.As. A late Release cannot affect a new token.
Release need not prove that business work has stopped; stop it first.

## Cleanup and shutdown

Call `manager.CleanupExpired(ctx, limit)` periodically with 1..10000. It deletes
at most limit still-expired rows per call, ordered through the expiry index.
Budget batches and time per sweep; do not run an unbounded cleanup loop. Custom
backends without CleanupBackend return ErrUnsupported. There are no implicit
background goroutines. See [PERFORMANCE.md](PERFORMANCE.md) for measurements,
hot-key backoff and dynamic-key disk planning.

Shutdown: stop admission, stop/join business tasks, release owned handles with a
bounded independent context, then Close the manager. Close is idempotent, waits
for active calls, and rejects new operations. It closes factory-owned pools;
external pools remain the caller's responsibility. Close alone neither revokes
database leases nor cancels business work.

## Offline upgrade / rollback

**Never run v1 and v2 participants together.** There is no online migration.

1. Stop all old participants AND their business tasks; verify paused processes
   cannot resume, revoke their credentials if needed.
2. Back up the old table/database and record its schema, engine, indexes and
   software version. Ensure `dlocks_v1_backup` does not already exist.
3. Run [migrations/v2.sql](migrations/v2.sql). RENAME preserves the old table;
   create an empty v2 table. Do not copy old leases into it. A fresh installation
   can simply invoke Initialize.
4. If interrupted, inspect table names: backup only means create the v2 table;
   both tables means validate the new table. Never rerun RENAME blindly, delete
   the backup, or start either version against an unknown schema.
5. Invoke Initialize. It validates column types, InnoDB, a full key-only unique
   index and expiry index by definition, including indexes with different names.
   It rejects incompatible tables; it never uses AutoMigrate or removes indexes.
6. Deploy only v2 clients; retain the backup for investigation. The key byte
   length limit and API are breaking changes requiring caller migration.

Rollback also requires stopping all v2 participants AND tasks. Keep the v2 table
under a distinct backup name, restore the old table name, clear old lease rows
while offline, and validate old schema before restarting old code. Never restore
a historical lease as valid ownership. Exercise upgrade/rollback on a disposable
copy first; production rollout acceptance remains the operator's responsibility.

## Tests

`go test -race ./dlock` and `go vet ./dlock` need no database. Examples compile.
Integration requires a **dedicated test database**; tests may delete expired rows
and temporary schema tests create/drop isolated databases. CI grants a test-only
account these privileges. Do not point tests at business databases.

```sh
TEST_MYSQL_HOST=127.0.0.1:3306 TEST_MYSQL_USER=root \
TEST_MYSQL_PWD=integration TEST_MYSQL_DB=dlock_test \
go test -race -tags integration -count=1 -timeout=180s ./dlock
```

For a local socket use TEST_MYSQL_NET=unix and set HOST to the socket path.
The integration entry fails if its configuration is missing; it never silently
skips. CI runs actual MySQL 8.0 and 8.4, race detection, and vet. Historical audit
probes under testdata describe the old vulnerability, not passing safety tests.
See [REMEDIATION.md](REMEDIATION.md) for closure and evidence.
