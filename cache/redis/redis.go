package redis

import (
	"context"
	"time"

	"github.com/beatlabs/patron/trace/redis"
)

// Cache encapsulates a Redis-based caching mechanism,
// driven by go-redis/redis/v7.
type Cache struct {
	rdb *redis.Conn
	ctx context.Context
}

// New returns a new Redis client that will be used as the cache store.
func New(opt redis.Options) (*Cache, error) {
	redisConn := redis.New(context.Background(), opt)
	return &Cache{rdb: redisConn, ctx: context.Background()}, nil
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
func (c *Cache) Set(key, value string) (interface{}, error) {
	return c.rdb.Do(c.ctx, "set", key, value)
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
func (c *Cache) SetTTL(key string, value interface{}, ttl time.Duration) (interface{}, error) {
	return c.rdb.Do(c.ctx, "set", key, value, "px", int(ttl.Milliseconds()))
}
