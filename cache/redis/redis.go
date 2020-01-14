package redis

import (
	"time"

	"github.com/go-redis/redis/v7"
)

type cache struct {
	redis.Client
}

func Create() (*cache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

}

func (c *cache) Contains(key string) bool                                {}
func (c *cache) Get(key string) (interface{}, bool, error)               {}
func (c *cache) Purge() error                                            {}
func (c *cache) Set(key string, value interface{})                       {}
func (c *cache) SetTTL(key string, value interface{}, ttl time.Duration) {}
