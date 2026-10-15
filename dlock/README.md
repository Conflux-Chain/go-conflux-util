# Distributed Lock

DLock coordinates access to shared resources through lease-based distributed
locks. It provides a MySQL backend and a `Backend` interface for custom storage
implementations.

A lock is identified by a key. A nonce identifies its holder for one holding
period, and a lease determines when another holder may acquire it. Calling
`Acquire` again with the same intent renews the lease; `Release` relinquishes it.

## Usage

Configure a MySQL database through the project's viper configuration:

```yaml
store:
  mysql:
    host: "127.0.0.1:3306"
    username: "app"
    password: "your-password"
    database: "app"
```

Initialize the database and lock table at the application's entry point using
`store.Config.OpenOrCreate`. The application owns the connection pool and closes
it after all database users have stopped. Pass the initialized database to DLock:

```go
config := store.MustNewConfigFromViper()
db, err := config.OpenOrCreate(&dlock.Dlock{})
if err != nil {
    return err
}
defer store.NewStore(db).Close()

ctx := context.Background()
backend := dlock.NewMySQLBackend(db)
manager := dlock.NewLockManager(backend)
```

The `Dlock` model defines the table structure through GORM tags. DLock performs
lock operations; database opening, table creation and connection pool management
belong to the application.

Generate a nonce and check acquisition before starting protected work. These
snippets assume an enclosing function returning `error`; see
[the complete example](example_test.go) for imports and error handling.

```go
var nonce [16]byte
if _, err := rand.Read(nonce[:]); err != nil {
    return err
}
intent := dlock.NewLockIntent("dlock_key", fmt.Sprintf("%x", nonce), 15*time.Second)
deadline, err := manager.AcquireWithDeadline(ctx, intent)
if err != nil {
    return err
}
defer func() {
    releaseCtx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()
    if err := manager.Release(releaseCtx, intent); err != nil {
        log.Printf("release lock: %v", err)
    }
}()

workCtx, cancelWork := context.WithDeadline(ctx, deadline)
defer cancelWork()
// Pass workCtx to protected operations and stop by deadline.
if err := workCtx.Err(); err != nil {
    return err
}

// Renew before expiry using the original context and the same intent.
deadline, err = manager.AcquireWithDeadline(ctx, intent)
if err != nil {
    return err
}
cancelWork()
workCtx, cancelWork = context.WithDeadline(ctx, deadline)
defer cancelWork()
if err := workCtx.Err(); err != nil {
    return err
}
// Continue protected work using the renewed workCtx.
```

`AcquireWithDeadline` returns a conservative local deadline, measured from before
storage access with Go's monotonic clock. Acquisition time and a 1ms precision
reserve reduce the usable lease. `Acquire` applies the same checks and returns
only an error. Both return `ErrLockAcquisitionFailed` on contention or exhausted
lease budget, including leases of 1ms or less. Failed acquisitions must never be
used for protected work. Cleanup uses key+nonce; cleanup errors remain observable
with `errors.Is`.
`Release` returns `ErrLockNotHeld` when no lock matches the key and nonce. Other
errors retain the underlying storage or context error and support `errors.Is`.

For custom storage, implement the `Backend` interface and pass it to
`dlock.NewLockManager(backend)`. The application manages the backend's resources.

## Usage notes

- Generate a unique, nonempty nonce for each holding period. Reuse it only for
  renewals; use a new nonce after release or lease loss. Do not share nonces
  between instances. Serialize acquisition, renewal and release for each intent.
- MySQL keys and nonces compare by bytes, including case and trailing spaces.
  Keys may contain at most 764 bytes.
- Leases must be positive and are rounded up to milliseconds. Timestamps are
  stored and compared in UTC using the MySQL server's clock, independently of
  connection time zones. Local deadlines assume a stable database clock; forward
  clock jumps can expire a lease earlier. Use a pool in autocommit mode, outside
  caller-managed transactions, so success includes committing the lock write.
- Renewals are explicit. Schedule them before expiry with sufficient margin for
  database delays, and stop protected work if renewal fails or lease validity is
  uncertain. If acquisition returns an error, do not start protected work.
- A paused holder may resume after its lease expires and another holder acquires
  the lock. If stale writes must be rejected, the protected resource must validate
  monotonically increasing fencing tokens; DLock does not provide these tokens.
- Acquisition and renewal normally require at least two SQL round trips. Size
  the connection pool and renewal frequency for expected contention.
- Expired records are not automatically removed. For workloads with continually
  changing keys, arrange cleanup that deletes only records still expired at
  deletion time, without removing renewed leases.
