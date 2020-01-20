package lru

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {

	c, err := New(-1)
	assert.Nil(t, c)
	assert.Error(t, err)

	c, err = New(512)
	assert.NotNil(t, c)
	assert.NoError(t, err)
}

func TestCacheOperations(t *testing.T) {
	c, err := New(10)
	assert.NotNil(t, c)
	assert.NoError(t, err)

	k, v := "foo", "bar"
	res, exists, err := c.Get(k)
	assert.Nil(t, res)
	assert.False(t, exists)
	assert.NoError(t, err)

	err = c.Set(k, v)
	assert.NoError(t, err)

	res, exists, err = c.Get(k)
	assert.Equal(t, v, res)
	assert.True(t, exists)
	assert.NoError(t, err)

	err = c.Remove(k)
	assert.NoError(t, err)
	res, exists, err = c.Get(k)
	assert.Nil(t, res)
	assert.False(t, exists)
	assert.NoError(t, err)

	err = c.Set("key1", "val1")
	assert.NoError(t, err)
	err = c.Set("key2", "val2")
	assert.NoError(t, err)
	err = c.Set("key3", "val3")
	assert.NoError(t, err)

	assert.Equal(t, c.lru.Len(), 3)
	err = c.Purge()
	assert.NoError(t, err)
	assert.Equal(t, c.lru.Len(), 0)

	err = c.SetTTL(k, v, 500*time.Millisecond)
	assert.NoError(t, err)
	res, exists, err = c.Get(k)
	assert.Equal(t, v, res)
	assert.True(t, exists)
	assert.NoError(t, err)

	time.Sleep(600 * time.Millisecond)
	res, exists, err = c.Get(k)
	assert.Nil(t, res)
	assert.False(t, exists)
	assert.NoError(t, err)
}
