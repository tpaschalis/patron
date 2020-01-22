package redis

import (
	"context"
	"time"

	"github.com/beatlabs/patron/trace/redis"
)

// Cache encapsulates a Redis-based caching mechanism,
// driven by go-redis/redis/v7.
type Cache struct {
	rdb *redis.Client
	ctx context.Context
}

// Options exposes the options struct from go-redis package
type Options redis.Options

// New returns a new Redis client that will be used as the cache store.
func New(opt Options) (*Cache, error) {
	redisDB := redis.New(context.Background(), redis.Options(opt))
	return &Cache{rdb: redisDB, ctx: context.Background()}, nil
}

// Get executes a lookup and returns whether a key exists in the cache along with and its value.
func (c *Cache) Get(key string) (interface{}, bool, error) {
	res, err := c.rdb.Do(c.ctx, "get", key)
	if err == redis.Empty || err != nil {
		return nil, false, err
	}
	return res, true, nil
}

// Set registers a key-value pair to the cache.
func (c *Cache) Set(key string, value interface{}) error {
	_, err := c.rdb.Do(c.ctx, "set", key, value)
	return err
}

// Purge evicts all keys present in the cache.
func (c *Cache) Purge() error {
	_, err := c.rdb.Do(c.ctx, "flushdb")
	return err
}

// Remove evicts a specific key from the cache.
func (c *Cache) Remove(key string) error {
	_, err := c.rdb.Do(c.ctx, "del", key)
	return err
}

// SetTTL registers a key-value pair to the cache. Once the provided duration expires,
// the function will try to erase the key from the cache.
func (c *Cache) SetTTL(key string, value interface{}, ttl time.Duration) error {
	_, err := c.rdb.Do(c.ctx, "set", key, value, "px", int(ttl.Milliseconds()))
	return err
}
