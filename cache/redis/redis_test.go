package redis

import (
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/beatlabs/patron/trace/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	opt := redis.Options{Addr: mr.Addr()}

	c, err := New(opt)
	assert.NotNil(t, c)
	assert.NoError(t, err)
}

func TestCacheOperationsMiniredis(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	opt := redis.Options{Addr: mr.Addr()}
	c, err := New(opt)
	k, v := "foo", "bar"

	t.Run("testSet", func(t *testing.T) {
		ok, err := c.Set(k, v)
		assert.Equal(t, "OK", ok)
		assert.NoError(t, err)
	})

	t.Run("testGet", func(t *testing.T) {
		res, ok, err := c.Get(k)
		assert.True(t, ok)
		assert.NoError(t, err)
		assert.Equal(t, v, res)

	})

	t.Run("testRemove", func(t *testing.T) {
		err := c.Remove(k)
		res, ok, err := c.Get(k)
		fmt.Println(res, ok, err)
		assert.False(t, ok)
		assert.Equal(t, redis.Empty, err)
	})

	t.Run("testPurge", func(t *testing.T) {
		_, err := c.Set("key1", "val1")
		assert.NoError(t, err)
		_, err = c.Set("key2", "val2")
		assert.NoError(t, err)
		_, err = c.Set("key3", "val3")
		assert.NoError(t, err)
		assert.Equal(t, int64(3), c.rdb.Client.DBSize().Val())

		err = c.Purge()
		assert.NoError(t, err)
		assert.Equal(t, int64(0), c.rdb.Client.DBSize().Val())
	})

	t.Run("testSetTTL", func(t *testing.T) {
		_, err := c.SetTTL(k, v, 500*time.Millisecond)
		assert.NoError(t, err)

		res, ok, err := c.Get(k)
		assert.Equal(t, v, res)
		assert.True(t, ok)
		assert.NoError(t, err)
		// miniredis doesn't decrease ttl automatically.
		mr.FastForward(500 * time.Millisecond)

		res, ok, err = c.Get(k)
		assert.Nil(t, res)
		assert.False(t, ok)
		assert.Equal(t, redis.Empty, err)

	})
}
