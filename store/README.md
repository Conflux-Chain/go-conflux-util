# Store Utilities

This package provides utilities to initialize [gorm](https://gorm.io/) in common, and also provides some very helpful `CRUD` utilities.

## Supported Databases

Currently, there are 3 types supported:

- Mysql: usually for prod mode.
- Sqlite (file): usually for test mode.
- Sqlite (memory): especially for unit test.

## Initialize gorm

It is highly recommended to initialize `gorm` from configuration file, see examples below.

```go
package main

import "github.com/Conflux-Chain/go-conflux-util/store"

func main() {
	// load config from configuration file
	config := store.MustNewConfigFromViper()

	// create database/tables or open database with table migrations
	db := config.MustOpenOrCreate(tables...)

    // create CRUD utilities
	store := store.NewStore(db)
	defer store.Close()
}
```

Generally, the database will be created if absent, and auto migrate all specified tables, e.g. add new column, index.

In unit test, you could initialize a memory database as following:

```go
config := store.NewMemoryConfig()
db := config.MustOpenOrCreate(tables...)
```

Use `NewMemoryConfig` for an in-memory database. It limits the pool to one open
connection, retains one idle connection, and disables connection lifetime expiry
to preserve the database across operations. If you construct or change the config
yourself, you are responsible for suitable connection pool settings. This also
applies to the automatic memory fallback when neither MySQL nor SQLite is configured;
`OpenOrCreate` uses the supplied pool settings as-is.

With `NewMemoryConfig`, operations share one connection. Within a transaction,
use its `tx` handle for database work; using the original `db` while the transaction
holds the connection can block waiting for that same connection. Close the pool when
finished, for example with `defer store.NewStore(db).Close()`.

## Store Utilities

There are some very helpful `CRUD` utilities available in [Store](./store.go) struct.
