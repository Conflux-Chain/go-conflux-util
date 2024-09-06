package parallel

import (
	"context"
	"testing"

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
