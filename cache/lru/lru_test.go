package lru

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreate(t *testing.T) {

	c, err := Create(-1)
	assert.Nil(t, c)
	assert.Error(t, err)

	c, err = Create(512)
	assert.NotNil(t, c)
	assert.NoError(t, err)
}

func TestCacheProperties(t *testing.T) {
	c, err := Create(10)
	assert.NotNil(t, c)
	assert.NoError(t, err)

	k, v := "foo", "bar"
	exists := c.Contains(k)
	assert.False(t, exists)

	err = c.Set(k, v)
	assert.NoError(t, err)

	res, exists, err := c.Get(k)
	assert.Equal(t, res, v)
	assert.True(t, exists)
	assert.NoError(t, err)

	err = c.Remove(k)
	exists = c.Contains(k)
	assert.NoError(t, err)
	assert.False(t, exists)

	c.Set("key1", "val1")
	c.Set("key2", "val2")
	c.Set("key3", "val3")
	assert.Equal(t, c.lru.Len(), 3)
	err = c.Purge()
	assert.NoError(t, err)
	assert.Equal(t, c.lru.Len(), 0)

	err = c.SetTTL(k, v, 500*time.Millisecond)
	assert.NoError(t, err)
	exists = c.Contains(k)
	assert.True(t, exists)
	time.Sleep(500 * time.Millisecond)
	exists = c.Contains(k)
	assert.False(t, exists)
}
