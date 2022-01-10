package redis

import (
	"context"
	"strings"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/store"
	viperutil "github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// RedisConfig represents the Redis configurations to open a Redis instance.
type RedisConfig struct {
	// connection DSN
	Dsn string
	// default cache expiration duration
	DefaultExpiration time.Duration
}

// RedisStore persists data in Redis cache.
type RedisStore struct {
	*redis.Client

	ctx context.Context
	// default cache expiration duration
	defaultDuration time.Duration
}

// MustNewStoreFromViper creates an instance of Redis store from viper
// or panic on error.
func MustNewStoreFromViper(ctx context.Context) *RedisStore {
	store, err := NewStoreFromViper(ctx)
	if err != nil {
		logrus.Fatal("Failed to create Redis store from viper")
	}

	return store
}

// NewStoreFromViper creates an instance of Redis store from viper.
func NewStoreFromViper(ctx context.Context) (*RedisStore, error) {
	var config RedisConfig

	err := viperutil.UnmarshalKey("store.redis", &config)
	if err != nil {
		err = errors.WithMessage(err, "failed to unmarshal config from viper")
		return nil, err
	}

	redisOpt, err := redis.ParseURL(config.Dsn)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to parse Redis url")
	}

	rdb := redis.NewClient(redisOpt)
	// Test redis connection
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return nil, errors.WithMessage(err, "unable to ping Redis server")
	}

	store := &RedisStore{
		Client: rdb, ctx: ctx, defaultDuration: config.DefaultExpiration,
	}

	return store, nil
}

// RedisKey returns a unified redis key sperated by colon
func RedisKey(keyParts ...string) string {
	return strings.Join(keyParts, ":")
}

// IsRecordNotFound implements Store interface method.
func (rs *RedisStore) IsRecordNotFound(err error) bool {
	return err == redis.Nil || err == store.ErrRecordNotFound
}

// Close implements Store interface method.
func (rs *RedisStore) Close() error {
	return rs.Client.Close()
}

// IncrBy implements CacheStore interface method.
//
// Increment the number stored at key by increment. If the key does not exist,
// it is set to 0 before performing the operation. An error is returned if the key
// contains a value of the wrong type or contains a string that can not be represented
// as integer. This operation is limited to 64 bit signed integers.
func (rs *RedisStore) IncrBy(k string, n int64) (int64, error) {
	return rs.Client.IncrBy(rs.ctx, k, n).Result()
}

// Get implements CacheStore interface method.
//
// Get the value of key. If the key does not exist nil is returned.
func (rs *RedisStore) Get(k string) (interface{}, bool, error) {
	v, err := rs.Client.Get(rs.ctx, k).Result()

	if rs.IsRecordNotFound(err) {
		return nil, false, nil
	}

	if err != nil {
		return nil, false, err
	}

	return v, true, nil
}

// Set implements CacheStore interface method.
//
// Set adds an item to the cache, replacing any existing item. If the duration is 0
// (DefaultExpiration), the cache's default expiration time is used. If it is -1
// (NoExpiration), the item never expires.
func (rs *RedisStore) Set(k string, v interface{}, d time.Duration) error {
	switch {
	case d == 0: // DefaultExpiration
		d = rs.defaultDuration
	case d < 0: // NoExpiration
		d = 0
	}

	_, err := rs.Client.Set(rs.ctx, k, v, d).Result()
	return err
}

// Delete implements CacheStore interface method.
//
// Delete an item from the cache. Does nothing if the key is not in the cache.
func (rs *RedisStore) Delete(k string) error {
	_, err := rs.Client.Del(rs.ctx, k).Result()
	return err
}

// Flush implements CacheStore interface method.
//
// Delete all the keys of all the existing databases, not just the currently
// selected one. This command never fails.
func (rs *RedisStore) Flush() error {
	return rs.Client.FlushDBAsync(rs.ctx).Err()
}
