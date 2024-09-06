package parallel

import (
	"context"
)

// Result represents a task execution result.
type Result[T any] struct {
	Routine int   // routine index
	Task    int   // task index
	Value   T     // task execution result
	err     error // task execution error
}

// Interface defines methods to support parallel execution.
type Interface[T any] interface {
	// ParallelDo executes task within given routine in parallel.
	//
	// Note, this method is thread unsafe.
	ParallelDo(ctx context.Context, routine, task int) (T, error)

	// ParallelCollect handles execution result in a separate goroutine.
	//
	// Note, this method is thread safe.
	ParallelCollect(ctx context.Context, result *Result[T]) error
}
