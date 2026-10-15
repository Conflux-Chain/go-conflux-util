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

After loading the configuration, create a manager. It creates the `dlocks` table
and owns its database connection pool:

```go
lockMan, err := dlock.OpenLockManagerFromViper()
if err != nil {
    return err
}
defer lockMan.Close()
```

`NewLockManagerFromConfig(conf)` accepts a `store.Config` directly.
`NewLockManagerFromViper()` is also available if initialization failures should
exit the process.

Generate a nonce and check acquisition before starting protected work. These
snippets assume an enclosing function returning `error`; see
[the complete example](example_test.go) for imports and error handling.

```go
var nonce [16]byte
if _, err := rand.Read(nonce[:]); err != nil {
    return err
}
intent := dlock.NewLockIntent("dlock_key", fmt.Sprintf("%x", nonce), 15*time.Second)
if err := lockMan.Acquire(ctx, intent); err != nil {
    return err
}
defer func() {
    releaseCtx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()
    if err := lockMan.Release(releaseCtx, intent); err != nil {
        log.Printf("release lock: %v", err)
    }
}()

// Perform bounded, cancellable work while the lease is valid.
// Renew before expiry with the same intent; stop work if renewal fails.
if err := lockMan.Acquire(ctx, intent); err != nil {
    return err
}
```

`Acquire` returns `ErrLockAcquisitionFailed` when another holder owns the lock.
`Release` returns `ErrLockNotHeld` when no lock matches the key and nonce. Other
errors retain the underlying storage or context error and support `errors.Is`.

To use a caller-owned backend, construct the manager with
`dlock.NewLockManager(backend)`. Its resources remain under caller ownership;
closing the manager does not close them. For caller-owned MySQL connections,
create the `dlock.Dlock` table with GORM, construct the backend with
`dlock.NewMySQLBackend(db)`, and call `backend.ValidateSchema(ctx)` before passing
it to `NewLockManager`.

## Usage notes

- Generate a unique, nonempty nonce for each holding period. Reuse it only for
  renewals; use a new nonce after release or lease loss. Do not share nonces
  between instances.
- MySQL keys and nonces compare by bytes, including case and trailing spaces.
  Keys may contain at most 764 bytes.
- Leases must be positive and are rounded up to milliseconds. Timestamps are
  stored and compared in UTC using the MySQL server's clock, independently of
  connection time zones. Client clock readings are not authoritative.
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
