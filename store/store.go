package store

import (
	"errors"

	"gorm.io/gorm"
)

type Store struct {
	DB *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db}
}

func (store *Store) Close() error {
	if db, err := store.DB.DB(); err != nil {
		return err
	} else {
		return db.Close()
	}
}

func (store *Store) Get(modelPtr interface{}, whereQuery string, args ...interface{}) (bool, error) {
	err := store.DB.Where(whereQuery, args...).First(modelPtr).Error
	if err == nil {
		return true, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}

	return false, err
}

func (store *Store) GetById(modelPtr interface{}, id uint64) (bool, error) {
	return store.Get(modelPtr, "id = ?", id)
}

// List queries a page ordered by ID. On success, it replaces the slice contents;
// an empty page leaves an empty slice. slicePtr must point to a slice.
func (store *Store) List(db *gorm.DB, idDesc bool, offset, limit int, slicePtr interface{}) (total int64, err error) {
	var orderBy string
	if idDesc {
		orderBy = "id DESC"
	} else {
		orderBy = "id ASC"
	}

	return store.ListByOrder(db, orderBy, offset, limit, slicePtr)
}

// ListByOrder queries a page with the given order. On success, it replaces the
// slice contents; an empty page leaves an empty slice. slicePtr must point to a slice.
func (*Store) ListByOrder(db *gorm.DB, orderBy string, offset, limit int, slicePtr interface{}) (total int64, err error) {
	if err = db.Count(&total).Error; err != nil {
		return 0, err
	}

	if err = db.Order(orderBy).Offset(offset).Limit(limit).Find(slicePtr).Error; err != nil {
		return 0, err
	}

	return total, nil
}
