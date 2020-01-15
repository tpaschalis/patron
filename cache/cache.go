package cache

import (
	"time"
)

// Cache interface for handling common operations, to be carried out by different caching implementations.
type Cache interface {
	Contains(key string) bool
	Get(key string) (interface{}, bool, error)
	Purge()
	Remove(key string)
	Set(key string, value interface{}) error
	SetTTL(key string, value interface{}, ttl time.Duration) error
}
