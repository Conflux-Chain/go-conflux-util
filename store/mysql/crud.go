package mysql

import (
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Crud wraps basic CRUD operations.
type Crud struct {
	store                  *MysqlStore // backend store
	errEntityNotFound      error       // not found error
	errEntityAlreadyExists error       // already exists error
}

// NewCrud constructs a Crud instance.
func NewCrud(store *MysqlStore, errEntityNotFound, errEntityAlreadyExists error) *Crud {
	return &Crud{store, errEntityNotFound, errEntityAlreadyExists}
}

// Exists checks if result set for specified where query (with args) existed or not.
func (crud *Crud) Exists(
	modelPtr interface{}, whereQuery string, args ...interface{},
) (bool, error) {
	err := crud.store.Where(whereQuery, args...).First(modelPtr).Error
	if err == nil {
		return true, nil
	}

	if crud.store.IsRecordNotFound(err) {
		return false, nil
	}

	return false, err
}

// Get gets result set for specified where query (with args).
func (crud *Crud) Get(
	modelPtr interface{}, whereQuery string, args ...interface{},
) error {
	exists, err := crud.Exists(modelPtr, whereQuery, args...)
	if err != nil {
		return err
	}

	if !exists {
		return crud.errEntityNotFound
	}

	return nil
}

// GetById gets result set by primary key ID.
func (crud *Crud) GetById(modelPtr interface{}, id uint64) error {
	return crud.Get(modelPtr, "id = ?", id)
}

// RequireAbsent requires record set for specified where query (with args) not existed,
// otherwise an entity already exists error will be returned.
func (crud *Crud) RequireAbsent(
	modelPtr interface{}, whereQuery string, args ...interface{},
) error {
	exists, err := crud.Exists(modelPtr, whereQuery, args...)
	if err != nil {
		return err
	}

	if exists {
		return crud.errEntityAlreadyExists
	}

	return nil
}

// List lists record sets ordered by primary key Id ascendantly or descendantly
// and pagination support. Extra query conditions could be passed in by
// constructing a gorm.DB instance.
func (crud *Crud) List(
	db *gorm.DB, idDESC bool, offset, limit int, destSlice interface{},
) (total int64, err error) {
	var orderBy string

	if idDESC {
		orderBy = "id DESC"
	} else {
		orderBy = "id ASC"
	}

	return crud.ListByOrder(db, orderBy, offset, limit, destSlice)
}

// ListByOrder lists record sets with (offset, limit) pagination support
// and specified orders. Extra query conditions could be passed in by
// constructing a gorm.DB instance.
func (*Crud) ListByOrder(
	db *gorm.DB, orderBy string, offset, limit int, destSlice interface{},
) (total int64, err error) {
	if err = db.Count(&total).Error; err != nil { // no records?
		return 0, err
	}

	if !logrus.IsLevelEnabled(logrus.TraceLevel) &&
		(total == 0 || total <= int64(offset)) {
		return total, nil
	}

	err = db.Order(orderBy).Offset(offset).Limit(limit).Find(destSlice).Error
	if err != nil {
		return 0, err
	}

	return total, nil
}
