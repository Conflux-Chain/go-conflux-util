# Distributed Lock

The distributed lock ensures atomicity in a distributed environment, such as leader election for achieving high availability.

To create a distributed lock, you need to specify a storage backend. We provide the `MySQLBackend` which handles reading and writing lock information in a MySQL table. Alternatively, you can implement your own storage backend using Redis, etcd, ZooKeeper, or other options.

```go
// Construct a lock manager with customized storage backend.
lockMan := dlock.NewLockManager(backend)
```

Alternatively, you can construct a lock manager with a convenient MySQL backend by using configuration files or environment variables.

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
