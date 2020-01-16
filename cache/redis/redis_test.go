package redis

import (
	"testing"
	"time"

	"github.com/beatlabs/patron/log"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v7"
	"github.com/stretchr/testify/assert"
)

func TestCreate(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		log.Fatal(err)
	}
	opt := redis.Options{Addr: mr.Addr()}

	c, err := Create(opt)
	assert.NotNil(t, c)
	assert.NoError(t, err)
}

func TestCacheOperationsMiniredis(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		log.Fatal(err)
	}
	opt := redis.Options{Addr: mr.Addr()}

	c, err := Create(opt)
	assert.NotNil(t, c)
	assert.NoError(t, err)

	k, v := "foo", "bar"
	exists, err := c.Contains(k)
	assert.NoError(t, err)
	assert.False(t, exists)

	err = c.Set(k, v)
	assert.NoError(t, err)

	res, exists, err := c.Get(k)
	assert.Equal(t, res, v)
	assert.True(t, exists)
	assert.NoError(t, err)

	err = c.Remove(k)
	assert.NoError(t, err)
	exists, err = c.Contains(k)
	assert.NoError(t, err)
	assert.False(t, exists)

	err = c.Set("key1", "val1")
	assert.NoError(t, err)
	err = c.Set("key2", "val2")
	assert.NoError(t, err)
	err = c.Set("key3", "val3")
	assert.NoError(t, err)

	assert.Equal(t, c.rdb.DBSize().Val(), int64(3))
	err = c.Purge()
	assert.NoError(t, err)
	assert.Equal(t, c.rdb.DBSize().Val(), int64(0))

	err = c.SetTTL(k, v, 500*time.Millisecond)
	assert.NoError(t, err)
	exists, err = c.Contains(k)
	assert.NoError(t, err)
	assert.True(t, exists)

	// miniredis doesn't decrease ttl automatically.
	mr.FastForward(500 * time.Millisecond)

	exists, err = c.Contains(k)
	assert.False(t, exists)
	assert.NoError(t, err)
}

func TestCacheOperationsMocked(t *testing.T) {
	c := NewMockRedis()

	k, v := "foo", "bar"
	exists, err := c.Contains(k)
	assert.NoError(t, err)
	assert.False(t, exists)

	err = c.Set(k, v)
	assert.NoError(t, err)

	res, exists, err := c.Get(k)
	assert.Equal(t, res, v)
	assert.True(t, exists)
	assert.NoError(t, err)

	err = c.Remove(k)
	assert.NoError(t, err)
	exists, err = c.Contains(k)
	assert.NoError(t, err)
	assert.False(t, exists)

	err = c.Set("key1", "val1")
	assert.NoError(t, err)
	err = c.Set("key2", "val2")
	assert.NoError(t, err)
	err = c.Set("key3", "val3")
	assert.NoError(t, err)

	assert.Equal(t, len(c.data), 3)
	err = c.Purge()
	assert.NoError(t, err)
	assert.Equal(t, len(c.data), 0)

	ttl := 500 * time.Millisecond
	err = c.SetTTL(k, v, ttl)
	assert.NoError(t, err)
	exists, err = c.Contains(k)
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, c.data[k].ttl, ttl)
}
