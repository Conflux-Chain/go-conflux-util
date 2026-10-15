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

## MySQL schema and upgrades

MySQL keys and nonces compare by bytes, including case, accents and trailing
spaces. Keys may contain at most 764 bytes; nonces must be nonempty. New tables
use `VARBINARY(764)` keys, `LONGBLOB` nonces and millisecond timestamps.
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
