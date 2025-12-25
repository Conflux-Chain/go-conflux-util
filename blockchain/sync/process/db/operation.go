package db

import (
	"reflect"

	"gorm.io/gorm"
)

// Processor is implemented by types that process data to update databases.
type Processor[T any] interface {
	Process(data T) Operation
}

// Operation represents a database operation that will be executed within a transaction.
type Operation interface {
	Exec(tx *gorm.DB) error
}

////////////////////////////////////////////////////////////////////////

type compositeOperation struct {
	ops []Operation
}

// ComposeOperation composes multiple database operations in batch.
func ComposeOperation(ops ...Operation) Operation {
	return compositeOperation{ops}
}

func (op compositeOperation) Exec(tx *gorm.DB) error {
	for _, v := range op.ops {
		if err := v.Exec(tx); err != nil {
			return err
		}
	}

	return nil
}

////////////////////////////////////////////////////////////////////////

type createOperation struct {
	models []any
}

// CreateOperation returns a Create database operation.
func CreateOperation(models ...any) Operation {
	return createOperation{models}
}

func (op createOperation) Exec(tx *gorm.DB) error {
	return tx.Create(op.models).Error
}

////////////////////////////////////////////////////////////////////////

type deleteOperation struct {
	modelPtr any // pointer type, e.g. &Foo{}
	conds    []any
}

// DeleteOperation returns a Delete database operation.
func DeleteOperation(modelPtr any, conds ...any) Operation {
	return deleteOperation{modelPtr, conds}
}

func (op deleteOperation) Exec(tx *gorm.DB) error {
	return tx.Delete(op.modelPtr, op.conds...).Error
}

////////////////////////////////////////////////////////////////////////

type batchOperation struct {
	creates         map[reflect.Type][]any
	createBatchSize int

	nonBatchableOps []Operation
}

func newBatchOperation(createBatchSize int) *batchOperation {
	return &batchOperation{
		creates:         make(map[reflect.Type][]any),
		createBatchSize: createBatchSize,
	}
}

func (batch *batchOperation) Add(ops ...Operation) {
	for _, v := range ops {
		switch op := v.(type) {
		case compositeOperation:
			batch.Add(op.ops...)
		case createOperation:
			if len(op.models) > 0 {
				modelType := reflect.TypeOf(op.models[0])
				batch.creates[modelType] = append(batch.creates[modelType], op.models...)
			}
		default:
			batch.nonBatchableOps = append(batch.nonBatchableOps, op)
		}
	}
}

func (batch *batchOperation) Exec(tx *gorm.DB) error {
	for _, v := range batch.creates {
		if err := tx.CreateInBatches(v, batch.createBatchSize).Error; err != nil {
			return err
		}
	}

	for _, v := range batch.nonBatchableOps {
		if err := v.Exec(tx); err != nil {
			return err
		}
	}

	return nil
}

func (batch *batchOperation) Rows() int {
	var rows int

	for _, v := range batch.creates {
		rows += len(v)
	}

	rows += len(batch.nonBatchableOps)

	return rows
}
