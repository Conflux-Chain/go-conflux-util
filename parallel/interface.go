package parallel

import (
	"context"
)

// Result represents a task execution result.
type Result[T any] struct {
	Routine int   // routine index
	Task    int   // task index
	Value   T     // task execution result
	Err     error // optional task execution error
}

// Interface defines methods to support parallel execution.
type Interface[T any] interface {
	// ParallelDo executes a task in a worker goroutine.
	//
	// Serial may call it concurrently on the same instance. Implementations must
	// synchronize shared state, including state also accessed by ParallelCollect.
	// The returned value and error are passed to ParallelCollect in Result.
	// Blocking operations should honor ctx so cancellation can stop the worker.
	ParallelDo(ctx context.Context, routine, task int) (T, error)

	// ParallelCollect handles results in task-index order in the goroutine calling Serial.
	//
	// Calls are serial within one Serial invocation, but may overlap with ParallelDo
	// or with collectors from other Serial invocations sharing the same instance.
	// It decides how to handle Result.Err; returning an error stops collection.
	// Blocking operations should honor ctx so cancellation can stop collection.
	ParallelCollect(ctx context.Context, result *Result[T]) error
}
