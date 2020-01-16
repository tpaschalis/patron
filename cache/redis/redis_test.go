package redis

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v7"
	"github.com/stretchr/testify/assert"
)

func TestCreate(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	_ = mr
	opt := redis.Options{
		Addr: mr.Addr(),
	}

	c, err := Create(opt)
	assert.NotNil(t, c)
	assert.NoError(t, err)
}

func TestCacheOperations(t *testing.T) {

}
