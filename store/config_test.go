package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memoryRecord struct {
	ID   uint64
	Name string
}

func TestNewMemoryConfig(t *testing.T) {
	config := NewMemoryConfig()
	assert.Equal(t, 1, config.MaxOpenConns)
	assert.Equal(t, 1, config.MaxIdleConns)
	assert.Zero(t, config.ConnMaxLifetime)

	db, err := config.OpenOrCreate(&memoryRecord{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, sqlDB.Close()) })
	assert.Equal(t, 1, sqlDB.Stats().MaxOpenConnections)

	// Separate operations reuse the retained connection and its in-memory data.
	require.NoError(t, db.Create(&memoryRecord{ID: 1, Name: "retained"}).Error)
	var record memoryRecord
	require.NoError(t, db.First(&record, 1).Error)
	assert.Equal(t, "retained", record.Name)
	assert.Equal(t, 1, sqlDB.Stats().Idle)
}
