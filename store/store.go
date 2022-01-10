package store

import (
	"errors"
	"io"
	"time"
)

var (
	// Store error - record not found
	ErrRecordNotFound = errors.New("record not found")
)

// Store is implemented by any object that persists data.
type Store interface {
	io.Closer

	// Check if record not found in store by error
	IsRecordNotFound(err error) bool
}

// CacheStore interface for data cache and store.
type CacheStore interface {
	Store

	// IncrBy increases an item of type int64 by n. If the key does not exist,
	// it is set to 0 before performing the operation. Returns an error if the
	// item's value is not an int64. If there is no error, the incremented
	// value is returned.
	IncrBy(k string, n int64) (int64, error)
	// Get gets an item from the cache. Returns the item or nil, and a bool indicating
	// whether the key was found.
	Get(k string) (interface{}, bool, error)
	// Set adds an item to the cache, replacing any existing item. If the duration is 0
	// (DefaultExpiration), the cache's default expiration time is used. If it is -1
	// (NoExpiration), the item never expires.
	Set(k string, v interface{}, d time.Duration) error
	// Delete an item from the cache. Does nothing if the key is not in the cache.
	Delete(k string) error
	// Flush deletes all items from the cache.
	Flush() error
}
