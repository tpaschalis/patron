package cache

import (
	"time"
)

// Cache interface for handling common operations, to be carried out by different caching implementations.
type Cache interface {
	Get(key string) (interface{}, bool, error)
	Purge() error
	Remove(key string) error
	Set(key string, value interface{}) error
}

// TTLCache interface for handling common operations, to be carried out by different caching implementations.
type TTLCache interface {
	Get(key string) (interface{}, bool, error)
	Purge() error
	Remove(key string) error
	SetTTL(key string, value interface{}, ttl time.Duration) error
}
