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

// Create returns a new Redis client that will be used for the cache
func Create() (*Cache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	return &Cache{rdb: rdb}, nil

}

// Contains returns whether the key exists in cache.
func (c *Cache) Contains(key string) (bool, error) {
	_, err := c.rdb.Get(key).Result()

	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// Get executes a lookup and returns whether a key exists in the cache along with and its value.
func (c *Cache) Get(key string) (interface{}, bool, error) {
	value, err := c.rdb.Get(key).Result()

	if err == redis.Nil {
		return nil, false, err
	}
	if err != nil {
		return nil, false, err
	}

	return value, true, nil
}

// Purge evicts all keys present in the cache.
func (c *Cache) Purge() {
	c.rdb.FlushDBAsync().Err()
}

// Remove evicts a specific key from the cache.
func (c *Cache) Remove(key string) {}

// Set registers a key-value pair to the cache.
func (c *Cache) Set(key string, value interface{}) error {
	err := c.rdb.Set(key, value, 0).Err()
	if err != nil {
		return err
	}
	return nil
}

// SetTTL registers a key-value pair to the cache. Once the provided duration expires,
// the function will try to erase the key from the cache.
func (c *Cache) SetTTL(key string, value interface{}, ttl time.Duration) error {
	err := c.rdb.Set(key, value, ttl).Err()
	if err != nil {
		return err
	}
	return nil
}
