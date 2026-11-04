package db_test

import (
	"testing"

	dbproc "github.com/Conflux-Chain/go-conflux-util/blockchain/sync/process/db"
	"github.com/Conflux-Chain/go-conflux-util/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type createRecord struct {
	ID   uint
	Name string
}

func newCreateTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	config := store.NewMemoryConfig()
	database, err := config.OpenOrCreate(&createRecord{})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, sqlDB.Close()) })
	return database
}

func TestCreateOperation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		models []any
		want   []createRecord
	}{
		{
			name:   "single model",
			models: []any{&createRecord{ID: 1, Name: "Alice"}},
			want:   []createRecord{{ID: 1, Name: "Alice"}},
		},
		{
			name:   "multiple models",
			models: []any{&createRecord{ID: 1, Name: "Alice"}, &createRecord{ID: 2, Name: "Bob"}},
			want:   []createRecord{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}},
		},
		{
			name:   "typed slice",
			models: []any{&[]createRecord{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}}},
			want:   []createRecord{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			database := newCreateTestDB(t)
			require.NoError(t, database.Transaction(func(tx *gorm.DB) error {
				return dbproc.CreateOperation(tc.models...).Exec(tx)
			}))
			var rows []createRecord
			require.NoError(t, database.Order("id").Find(&rows).Error)
			assert.Equal(t, tc.want, rows)
		})
	}
}

func TestCreateOperationRollback(t *testing.T) {
	database := newCreateTestDB(t)
	err := database.Transaction(func(tx *gorm.DB) error {
		err := dbproc.CreateOperation(
			&createRecord{ID: 1, Name: "Alice"},
			&createRecord{ID: 1, Name: "duplicate"},
		).Exec(tx)
		assert.Error(t, err)
		// The first insert succeeded before the duplicate key failed.
		var row createRecord
		if queryErr := tx.First(&row, 1).Error; assert.NoError(t, queryErr) {
			assert.Equal(t, "Alice", row.Name)
		}
		return err
	})
	require.Error(t, err)
	var count int64
	require.NoError(t, database.Model(&createRecord{}).Count(&count).Error)
	assert.Zero(t, count)
}
