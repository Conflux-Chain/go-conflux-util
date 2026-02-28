# Blockchain Data Sync

Many services require to synchronize blockchain data, e.g. blocks, transactions, receipts and traces. This package provides a data synchronization framework to resolve some problems in common:

- Pipeline: poll blockchain data, transform data and store data in separate goroutines.
- Concurrency: poll blockchain data in parallel during catch up phase.
- Batch: allow to store data into databases in batch, stead of block by block during catch up phase.
- Chain reorg: framework detects chain reorg and defines a common interface for clients to handle chain reorg (e.g. pop data from database).
- Memory bounded: uses memory bounded channel to cache polled blockchain data to void OOM.

Using this framework, users only need to focus on how to handle the data.

## Adapter

This package provides a [Adapter](./poll/adapter.go) interface to adapt any data source to poll blockchain data.

There are 2 pre-defined adapters:

1. [EVM adapter](./evm/adapter.go): poll data from eSpace RPC.
2. [Core adapter](./core/adapter.go): poll data from core space RPC.

## Poller

There are 3 kinds of pollers available:

1. [CatchUpPoller](./poll/catchup_poller.go): optimized to poll data in catch up phase with high performance.
2. [FinalizedPoller](./poll/finalized_poller.go): poll finalized data block by block.
3. [LatestPoller](./poll/latest_poller.go): poll latest data block by block, and handle the chain reorg.

## Database Processor

This package defines a common interface to transform blockchain data into a database operation, so that the framework will operate database in a transaction. Besides, some common used operations are already defined.

```go
type Operation interface {
    Exec(tx *gorm.DB) error
}

// ComposeOperation composes multiple database operations in batch.
func ComposeOperation(ops ...Operation) Operation

// CreateOperation returns a Create database operation.
func CreateOperation(models ...any) Operation

// DeleteOperation returns a Delete database operation.
func DeleteOperation(modelPtr any, conds ...any) Operation
```

User could implement below interface to transform the blockchain data into a database operation:

```go
// Processor is implemented by types that process data to update database.
type Processor[T any] interface {
    Process(data T) Operation
}
```

If user want to synchronize the latest data, and handle the chain reorg correctly. Then, please implements the revertable interface as below:

```go
// RevertableProcessor is implemented by types that process revertable data.
type RevertableProcessor[T any] interface {
    Processor[T]

    // Revert deletes data from database of given data block number.
    Revert(data T) Operation
}
```

During catch up phase, to achieve batch database operations, user could implement the batchable interface:

```go
// BatchProcessor is implemented by types that process data in batch.
//
// Note, thread-safe is not required in the implementations, since batch
// related methods are executed in a single thread.
type BatchProcessor[T any] interface {
    Processor[T]

    // BatchProcess processes the given data and returns the number of SQLs
    // to be executed in batch.
    BatchProcess(data T) int

    // BatchExec executes SQLs in batch.
    BatchExec(tx *gorm.DB, createBatchSize int) error

    // BatchReset reset data for the next batch.
    BatchReset()
}
```

## Sync Utilities

There are 3 helper methods available in the framework to poll blockchain data and store in database. Users need to provide custom database processors to handle polled blockchain data.

1. [CatchUpDB](./sync_db.go): catch up blockchain data to the finalized block using `BatchProcessor`.
2. [StartFinalizedDB](./sync_db.go): start to synchronize data block by block against the finalized block using normal `Processor`.
3. [StartLatestDB](./sync_db.go): start to synchronize data block by block against the latest block and handle chain reorg using `RevertableProcessor`.
