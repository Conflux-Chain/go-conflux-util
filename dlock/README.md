# Distributed Lock

The distributed lock ensures atomicity in a distributed environment, such as leader election for achieving high availability.

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
// Construct a lock manager with a MySQL backend from configurations loaded by viper
lockMan = dlock.NewLockManagerFromViper()
```

To acquire and release a lock, you can use:

```go
intent := NewLockIntent("dlock_key", 15 * time.Second)
// Acquire a lock with key name "dlock_key" for 15 seconds
lockMan.Acquire(context.Background(), intent)
// Release the lock immediately
lockMan.Release(context.Background(), intent)
```
