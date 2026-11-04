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
// Each argument is a model pointer or a typed slice supported by GORM.
//
// Examples (users is a []User):
//
//	CreateOperation(&User{Name: "Alice"})                    // single model
//	CreateOperation(&User{Name: "Alice"}, &Order{UserID: 1})  // multiple models
//	CreateOperation(&users)                                // batch of the same model type
//
// Pass a typed slice as one argument; do not convert it to []any and expand it
// if you want GORM to insert it as a batch.
//
// All arguments use the supplied transaction; the caller owns transaction handling.
func CreateOperation(models ...any) Operation {
	return createOperation{models}
}

func (op createOperation) Exec(tx *gorm.DB) error {
	for _, model := range op.models {
		if err := tx.Create(model).Error; err != nil {
			return err
		}
	}
	return nil
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
