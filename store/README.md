# Store Utilities

This package provides utilities to initialize [gorm](https://gorm.io/) in common, and also provides some very helpful `CRUD` utilities.

## Supported Databases

Currently, there are 3 types supported:

- Mysql: used for prod mode.
- Sqlite (file): used for test mode.
- Sqlite (memory): used for unit test.

## Initialize gorm

It is highly recommended to initialize gorm from configuration file:

```go
// load config from configuration file
config := store.MustNewConfigFromViper()
// create database/tables or open database with table migrations
db := config.MustOpenOrCreate(tables ...interface{})
```

In unit test, you could initialize a memory database as following:

```go
config := store.NewMemoryConfig()
db := config.MustOpenOrCreate(tables ...interface{})
```

## Store Utilities

There are some very helpful `CRUD` utilities available in [Store](./store.go) struct.