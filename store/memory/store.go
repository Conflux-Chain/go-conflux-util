package memory

import (
	"time"

	"github.com/Conflux-Chain/go-conflux-util/store"
	gocache "github.com/patrickmn/go-cache"
)

const (
	// purges expired items every 10 minutes
	defaultCleanupInterval time.Duration = 10 * time.Minute
)

// MemoryStore persists data in memory.
type MemoryStore struct {
	*gocache.Cache
}

// NewMemoryStore constructs an instance of MemoryStore. If clean up interval
// not specified with a non-zero value, a default clean up interval (10m) will
// be used.
func NewMemoryStore(defaultExpiration, defaultCleanup time.Duration) *MemoryStore {
	if defaultCleanup == 0 { // use default
		defaultCleanup = defaultCleanupInterval
	}

	return &MemoryStore{
		Cache: gocache.New(defaultExpiration, defaultCleanup),
	}
}

// IsRecordNotFound implements Store interface method.
func (s *MemoryStore) IsRecordNotFound(err error) bool {
	return err == store.ErrRecordNotFound
}

// Close implements Store interface method.
func (s *MemoryStore) Close() error {
	s.Cache.Flush()
	return nil
}

// IncrBy implements CacheStore interface method.
//
// Increment an item of type int64 by n. If the key does not exist, it is set
// to 0 before performing the operation. Returns an error if the item's value is
// not an int64,. If there is no error, the incremented value is returned.
func (s *MemoryStore) IncrBy(k string, n int64) (int64, error) {
	_, existed := s.Cache.Get(k)
	if !existed {
		s.Cache.Set(k, 0, 0)
	}

	return s.Cache.IncrementInt64(k, n)
}

// Get implements CacheStore interface method.
//
// Get the value of key. If the key does not exist nil is returned.
func (s *MemoryStore) Get(k string) (interface{}, bool, error) {
	v, existed := s.Cache.Get(k)
	return v, existed, nil
}

// Set implements CacheStore interface method.
//
// Set adds an item to the cache, replacing any existing item. If the duration is 0
// (DefaultExpiration), the cache's default expiration time is used. If it is -1
// (NoExpiration), the item never expires.
func (s *MemoryStore) Set(k string, v interface{}, d time.Duration) error {
	s.Cache.Set(k, v, d)
	return nil
}

// Delete implements CacheStore interface method.
//
// Delete an item from the cache. Does nothing if the key is not in the cache.
func (s *MemoryStore) Delete(k string) error {
	s.Cache.Delete(k)
	return nil
}

// Flush implements CacheStore interface method.
//
// Delete all items from the cache.
func (s *MemoryStore) Flush() error {
	s.Cache.Flush()
	return nil
}
