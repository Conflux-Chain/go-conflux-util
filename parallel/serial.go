package parallel

import (
	"context"
	"runtime"
	"sync"
)

type SerialOption struct {
	// Routines indicates the number of goroutines for parallel execution.
	//
	// By default, runtime.GOMAXPROCS(0) is used.
	Routines int

	// Window limits the maximum number of cached task execution result to handle in sequence.
	//
	// By default, 0 indicates no limit to execute all tasks as fast as it can.
	//
	// Otherwise, please set a proper value for some considerations, e.g.
	//  - Limits the memory usage to cache task execution results.
	//  - Business logic required.
	Window int
}

func (opt *SerialOption) Normalize(tasks int) {
	// 0 < routines <= tasks
	if opt.Routines <= 0 {
		opt.Routines = runtime.GOMAXPROCS(0)
	}

	if opt.Routines > tasks {
		opt.Routines = tasks
	}

	if opt.Window <= 0 {
		return
	}

	// routines <= window <= tasks
	if opt.Window < opt.Routines {
		opt.Window = opt.Routines
	}

	if opt.Window > tasks {
		opt.Window = tasks
	}
}

// Serial schedules tasks in sequence and handles the execution result in sequence as well.
// If any task execution failed, it will terminate and return error immediately.
func Serial[T any](ctx context.Context, parallelizable Interface[T], tasks int, option ...SerialOption) error {
	if tasks <= 0 {
		return nil
	}

	// normalize option
	var opt SerialOption
	if len(option) > 0 {
		opt = option[0]
	}
	opt.Normalize(tasks)

	// create channels
	chLen := max(opt.Routines, opt.Window)
	taskCh := make(chan int, chLen)
	defer close(taskCh)
	resultCh := make(chan *Result[T], chLen)
	defer close(resultCh)

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)

	// start goroutines to execute tasks
	for i := 0; i < opt.Routines; i++ {
		wg.Add(1)
		go doWork(ctx, parallelizable, i, taskCh, resultCh, &wg)
	}

	// fill task channel at first
	for i := 0; i < chLen; i++ {
		taskCh <- i
	}

	// collect execution results
	err := serialCollect(ctx, parallelizable, taskCh, resultCh, tasks, opt)

	// notify all goroutines to terminate
	cancel()

	// wait for terminations of all goroutines
	wg.Wait()

	return err
}

func doWork[T any](ctx context.Context, parallelizable Interface[T], routine int, taskCh <-chan int, resultCh chan<- *Result[T], wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case task := <-taskCh:
			val, err := parallelizable.ParallelDo(ctx, routine, task)

			resultCh <- &Result[T]{routine, task, val, err}

			// fail fast
			if err != nil {
				return
			}
		}
	}
}

func serialCollect[T any](ctx context.Context, parallelizable Interface[T], taskCh chan<- int, resultCh <-chan *Result[T], tasks int, opt SerialOption) error {
	nextTask := max(opt.Routines, opt.Window)
	cache := make(map[int]*Result[T])
	var nextResult int

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case result := <-resultCh:
			// fail fast
			if result.err != nil {
				return result.err
			}

			// immediately schedule new task if window disabled
			if opt.Window <= 0 && nextTask < tasks {
				taskCh <- nextTask
				nextTask++
			}

			cache[result.Task] = result

			// handle task in sequence
			for cache[nextResult] != nil {
				// schedule new task as soon as possible if window enabled
				if opt.Window > 0 && nextTask < tasks {
					taskCh <- nextTask
					nextTask++
				}

				if err := parallelizable.ParallelCollect(ctx, cache[nextResult]); err != nil {
					return err
				}

				// clear cache
				delete(cache, nextResult)
				nextResult++
			}

			// all tasks completed
			if nextResult >= tasks {
				return nil
			}
		}
	}
}
