package parallel

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type serialTester struct {
	t      *testing.T
	result []int
}

func (tester *serialTester) ParallelDo(ctx context.Context, routine, task int) (int, error) {
	return task * task, nil
}

func (tester *serialTester) ParallelCollect(ctx context.Context, result *Result[int]) error {
	assert.Nil(tester.t, result.Err)
	assert.Equal(tester.t, len(tester.result), result.Task)
	assert.Equal(tester.t, result.Task*result.Task, result.Value)

	tester.result = append(tester.result, result.Value)

	return nil
}

func TestSerialDefault(t *testing.T) {
	st := serialTester{t, nil}

	tasks := 100

	err := Serial(context.Background(), &st, tasks)
	assert.Nil(t, err)
	assert.Equal(t, tasks, len(st.result))

	for i := 0; i < tasks; i++ {
		assert.Equal(t, i*i, st.result[i])
	}
}

func TestSerialWindow(t *testing.T) {
	st := serialTester{t, nil}

	tasks := 100
	opt := SerialOption{
		Routines: 3,
		Window:   10,
	}

	err := Serial(context.Background(), &st, tasks, opt)
	assert.Nil(t, err)
	assert.Equal(t, tasks, len(st.result))

	for i := 0; i < tasks; i++ {
		assert.Equal(t, i*i, st.result[i])
	}
}

type serialCallbacks struct {
	do      func(context.Context, int) (int, error)
	collect func(context.Context, *Result[int]) error
}

func (callbacks serialCallbacks) ParallelDo(ctx context.Context, routine, task int) (int, error) {
	return callbacks.do(ctx, task)
}

func (callbacks serialCallbacks) ParallelCollect(ctx context.Context, result *Result[int]) error {
	return callbacks.collect(ctx, result)
}

func TestSerialTaskError(t *testing.T) {
	taskErr := errors.New("task failed")
	for _, tt := range []struct {
		name          string
		collectErr    error
		wantCollected int
	}{
		{name: "ignored by collector", wantCollected: 3},
		{name: "propagated by collector", collectErr: taskErr, wantCollected: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			collected := 0
			callbacks := serialCallbacks{
				do: func(_ context.Context, task int) (int, error) {
					return task, taskErr
				},
				collect: func(_ context.Context, result *Result[int]) error {
					assert.ErrorIs(t, result.Err, taskErr)
					assert.Equal(t, collected, result.Task)
					assert.Equal(t, result.Task, result.Value)
					collected++
					return tt.collectErr
				},
			}

			err := Serial(ctx, callbacks, 3, SerialOption{Routines: 1, Window: 1})
			assert.ErrorIs(t, err, tt.collectErr)
			assert.Equal(t, tt.wantCollected, collected)
			assert.NoError(t, ctx.Err())
		})
	}
}

func TestSerialCollectErrorCancelsWorker(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	collectErr := errors.New("collection failed")
	workerStarted := make(chan struct{})
	workerExited := make(chan struct{})
	var collected []int
	callbacks := serialCallbacks{
		do: func(ctx context.Context, task int) (int, error) {
			if task == 0 {
				// Ensure the other worker is running before collection fails.
				select {
				case <-workerStarted:
					return task, nil
				case <-ctx.Done():
					return 0, ctx.Err()
				}
			}
			close(workerStarted)
			defer close(workerExited)
			<-ctx.Done()
			return 0, ctx.Err()
		},
		collect: func(_ context.Context, result *Result[int]) error {
			assert.NoError(t, result.Err)
			collected = append(collected, result.Task)
			return collectErr
		},
	}

	err := Serial(ctx, callbacks, 2, SerialOption{Routines: 2, Window: 2})
	assert.ErrorIs(t, err, collectErr)
	assert.Equal(t, []int{0}, collected)
	assert.NoError(t, ctx.Err(), "worker should stop on internal cancellation, before the parent deadline")
	select {
	case <-workerExited:
	default:
		t.Error("Serial returned before the worker callback exited")
	}
}

func TestSerialContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	workerStarted := make(chan struct{})
	workerExited := make(chan struct{})
	callbacks := serialCallbacks{
		do: func(ctx context.Context, _ int) (int, error) {
			close(workerStarted)
			defer close(workerExited)
			<-ctx.Done()
			return 0, ctx.Err()
		},
		collect: func(ctx context.Context, _ *Result[int]) error {
			// A ready result may be selected even after cancellation.
			return ctx.Err()
		},
	}
	resultCh := make(chan error, 1)
	go func() {
		resultCh <- Serial(ctx, callbacks, 1, SerialOption{Routines: 1})
	}()

	select {
	case <-workerStarted:
	case <-ctx.Done():
		t.Fatal("Timed out waiting for the worker to start")
	}
	cancel()

	select {
	case err := <-resultCh:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for Serial to return after cancellation")
	}
	select {
	case <-workerExited:
	default:
		t.Error("Serial returned before the worker callback exited")
	}
}

func TestSerialNoTasks(t *testing.T) {
	for _, tt := range []struct {
		name  string
		tasks int
	}{
		{name: "zero", tasks: 0},
		{name: "negative", tasks: -1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			callbacks := serialCallbacks{
				do: func(context.Context, int) (int, error) {
					t.Error("ParallelDo called without tasks")
					return 0, nil
				},
				collect: func(context.Context, *Result[int]) error {
					t.Error("ParallelCollect called without tasks")
					return nil
				},
			}

			assert.NoError(t, Serial(ctx, callbacks, tt.tasks))
		})
	}
}
