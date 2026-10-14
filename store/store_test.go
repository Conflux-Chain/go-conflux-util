package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreListReplacesResults(t *testing.T) {
	for _, tc := range []struct {
		name   string
		seed   []memoryRecord
		offset int
		want   []memoryRecord
	}{
		{"empty table", nil, 0, []memoryRecord{}},
		{"offset at end", []memoryRecord{{ID: 1}, {ID: 2}}, 2, []memoryRecord{}},
		{"offset past end", []memoryRecord{{ID: 1}, {ID: 2}}, 3, []memoryRecord{}},
		{"nonempty page", []memoryRecord{{ID: 1}, {ID: 2}}, 1, []memoryRecord{{ID: 2}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := NewMemoryConfig()
			db, err := config.OpenOrCreate(&memoryRecord{})
			require.NoError(t, err)
			store := NewStore(db)
			t.Cleanup(func() { assert.NoError(t, store.Close()) })
			if len(tc.seed) > 0 {
				require.NoError(t, db.Create(&tc.seed).Error)
			}

			rows := []memoryRecord{{ID: 999, Name: "stale"}}
			total, err := store.List(db.Model(&memoryRecord{}), false, tc.offset, 1, &rows)
			require.NoError(t, err)
			assert.Equal(t, int64(len(tc.seed)), total)
			assert.Equal(t, tc.want, rows)
		})
	}
}
