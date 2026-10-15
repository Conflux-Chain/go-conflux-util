# Distributed Lock

This package coordinates lease-based ownership through a storage backend. It does
not make business operations atomic or guarantee that only one process continues
executing after a lease expires.

For example, holder A may pause until its lease expires, holder B may acquire the
lock, and A may then resume. Both can still operate on the protected resource.
Renew before expiry and stop protected work if renewal fails or lease validity is
uncertain; renewal cannot eliminate process pauses or undo in-flight operations.
After an ambiguous acquisition error, do not start protected work: the database
may have applied a write even if its response was lost.

If stale writes must be rejected, the protected resource must enforce a persistent,
monotonically increasing fencing token. This package does not provide one.
`Dlock.Version` is internal update bookkeeping: it is not returned by Acquire and
resets when a deleted row is recreated, so it cannot serve as a fencing token.

To create a distributed lock, you need to specify a storage backend. We provide the `MySQLBackend` which handles reading and writing lock information in a MySQL table. Alternatively, you can implement your own storage backend using Redis, etcd, ZooKeeper, or other options.

```go
// Construct a lock manager with customized storage backend.
lockMan := dlock.NewLockManager(backend)
```

Alternatively, you can construct a lock manager with a convenient MySQL backend by using configuration files or environment variables.

A nonce identifies one independent holding period. Generate a unique, nonempty
nonce for every acquisition after release or lease loss, and reuse it only for
renewals during that holding period. Do not share nonces between instances.
The package rejects empty nonces but cannot detect historical reuse: a delayed
release with a reused nonce could delete a later holder's lock.

```go
// Handle initialization errors and close the internally owned pool when done.
lockMan, err := dlock.OpenLockManagerFromViper()
if err != nil {
    return err
}
defer lockMan.Close()
```

The legacy `NewLockManagerFromViper()` exits the process on initialization failure.
`NewLockManagerFromConfig(conf)` accepts an existing `store.Config` and returns an
error. Managers created with `NewLockManager(backend)` leave backend resources
under caller ownership.

To acquire a lock, generate a nonce for this holding period and check the result
before starting protected work. These snippets assume an enclosing function
returning `error`; see [the compilable example](example_test.go) for imports and
complete error handling.

```go
var nonce [16]byte
if _, err := rand.Read(nonce[:]); err != nil {
    return err
}
intent := dlock.NewLockIntent("dlock_key", fmt.Sprintf("%x", nonce), 15*time.Second)
if err := lockMan.Acquire(ctx, intent); err != nil {
    return err
}
// Only arrange release after acquisition succeeds. Use a fresh context so that
// cancellation of the work does not prevent the release attempt.
defer func() {
    releaseCtx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()
    if err := lockMan.Release(releaseCtx, intent); err != nil {
        log.Printf("release lock: %v", err)
    }
}()

// Renew before expiry using the same intent; stop protected work on any error.
if err := lockMan.Acquire(ctx, intent); err != nil {
    return err
}
```

This package does not run a renewal goroutine. Keep work bounded and cancellable,
and choose a renewal schedule with sufficient margin for database delays. Once
released or lease validity is lost, use a new nonce for another holding period.

## MySQL schema and upgrades

MySQL keys and nonces compare by bytes, including case, accents and trailing
spaces. Keys may contain at most 764 bytes; nonces must be nonempty. New tables
use `VARBINARY(764)` keys, `LONGBLOB` nonces and millisecond timestamps.
Leases must be positive and are rounded up to whole milliseconds. Expiration is
measured using the MySQL server's millisecond clock; a lock is expired at its
deadline, including equality. Client clock readings are not authoritative.
Configuration constructors create missing tables but reject incompatible existing
tables. With a caller-owned database, create `Dlock` on a new database, or migrate
an existing table explicitly, then call `backend.ValidateSchema(ctx)` before use.

To upgrade, stop all old instances and their protected work, wait for outstanding
leases to expire, and back up the table. Do not mix old and new versions. Inspect
the actual schema, indexes and values before changing any columns:

```sql
SHOW FULL COLUMNS FROM dlocks;
SHOW INDEX FROM dlocks;
SELECT COUNT(*) AS invalid_rows FROM dlocks
WHERE OCTET_LENGTH(`key`) > 764 OR `key` IS NULL
   OR nonce IS NULL OR OCTET_LENGTH(nonce) = 0 OR expired_at IS NULL;
SELECT HEX(`key`), COUNT(*) FROM dlocks
GROUP BY HEX(`key`) HAVING COUNT(*) > 1;
```

Resolve invalid data first. The default old schema can be migrated with:

```sql
ALTER TABLE dlocks
  MODIFY `key` VARBINARY(764) NOT NULL,
  MODIFY nonce LONGBLOB NOT NULL,
  MODIFY expired_at DATETIME(3) NOT NULL,
  MODIFY created_at DATETIME(3) NULL,
  MODIFY updated_at DATETIME(3) NULL;
```

Retain the full unique `uidx_key` index on `key`. If it is missing, add
``CREATE UNIQUE INDEX uidx_key ON dlocks (`key`);`` after resolving duplicate data;
replace any conflicting prefix or composite index explicitly. Run schema
validation before restarting. Formerly folded identifiers are not recoverable
from the stored row; review application key names when changing comparison rules.

## Contention and capacity

Each acquisition or renewal reads a row before attempting a write, normally
requiring at least two SQL round trips; a competing holder may still cause an
unsuccessful update. Hot keys amplify database load, so size the pool and renewal
frequency for the expected contention. There is no background expiry cleanup:
expired rows remain until the key is reused or the matching nonce releases them.
Fixed sets of keys bound table growth, but continually changing keys can accumulate
rows after holders exit. If needed, arrange bounded database cleanup that deletes
only rows still expired at deletion time, without removing renewed leases. Atomic
conditional writes and automatic cleanup are not implemented by this package.
