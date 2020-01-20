package redis

import (
	"time"

	"github.com/go-redis/redis/v7"
)

// Cache encapsulates a Redis-based caching mechanism,
// driven by go-redis/redis/v7.
type Cache struct {
	rdb *redis.Client
}

// New returns a new Redis client that will be used as the cache store.
func New(opt redis.Options) (*Cache, error) {
	rdb := redis.NewClient(&opt)
	return &Cache{rdb: rdb}, nil
}

// Get executes a lookup and returns whether a key exists in the cache along with and its value.
func (c *Cache) Get(key string) (interface{}, bool, error) {
	value, err := c.rdb.Get(key).Result()

	if err != nil {
		return nil, false, err
	}

	return value, true, nil
}

// Purge evicts all keys present in the cache.
func (c *Cache) Purge() error {
	return c.rdb.FlushDBAsync().Err()
}

// Remove evicts a specific key from the cache.
func (c *Cache) Remove(key string) error {
	return c.rdb.Del(key).Err()
}

// Set registers a key-value pair to the cache.
func (c *Cache) Set(key string, value interface{}) error {
	return c.rdb.Set(key, value, 0).Err()
}

// SetTTL registers a key-value pair to the cache. Once the provided duration expires,
// the function will try to erase the key from the cache.
func (c *Cache) SetTTL(key string, value interface{}, ttl time.Duration) error {
	return c.rdb.Set(key, value, ttl).Err()
}
