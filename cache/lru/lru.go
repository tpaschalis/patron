package lru

import (
	"time"

	lru "github.com/hashicorp/golang-lru"
)

// Cache encapsulates a thread-safe fixed size LRU cache
// as defined in hashicorp/golang-lru.
type Cache struct {
	lru *lru.Cache
}

// Create returns a new LRU cache.
func Create(size int) (*Cache, error) {
	lruCache, err := lru.New(size)
	if err != nil {
		return nil, err
	}
	return &Cache{lru: lruCache}, nil
}

// Contains returns whether the key exists in cache, without updating its recent-ness.
func (c *Cache) Contains(key string) bool {
	return c.lru.Contains(key)
}

// Get executes a lookup and returns whether a key exists in the cache along with and its value.
func (c *Cache) Get(key string) (interface{}, bool, error) {
	value, ok := c.lru.Get(key)
	return value, ok, nil
}

// Purge evicts all keys present in the cache.
func (c *Cache) Purge() {
	c.lru.Purge()
}

// Remove evicts a specific key from the cache.
func (c *Cache) Remove(key string) {
	c.lru.Remove(key)
}

// Set registers a key-value pair to the cache.
func (c *Cache) Set(key string, value interface{}) error {
	c.lru.Add(key, value)

	return nil
}

// SetTTL registers a key-value pair to the cache. Once the provided duration expires,
// the function will try to erase the key from the cache.
func (c *Cache) SetTTL(key string, value interface{}, ttl time.Duration) errors {
	c.lru.Add(key, value)
	time.AfterFunc(ttl, func() {
		c.Remove(key)
	})

	return nil
}