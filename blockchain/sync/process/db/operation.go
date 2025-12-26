package db

import (
	"gorm.io/gorm"
)

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
