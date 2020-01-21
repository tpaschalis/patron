package redis

import (
	"context"
	"errors"
	"fmt"

	"github.com/beatlabs/patron/trace"
	"github.com/go-redis/redis/v7"
	"github.com/opentracing/opentracing-go"
)

type connInfo struct {
	instance string
}

func (c *connInfo) startSpan(ctx context.Context, opName, stmt string) (opentracing.Span, context.Context) {
	return trace.RedisSpan(ctx, opName, trace.RedisComponent, trace.RedisDBType, stmt, c.instance)
}

// Client wraps redis.Client for easier usage.
type Client redis.Client

// Options wraps redis.Options for easier usage.
type Options redis.Options

// Empty represents the error which is returned in case a key is not found.
const Empty = redis.Nil

// Conn represents a connection with a Redis client.
type Conn struct {
	connInfo
	Client *redis.Client
}

// New returns a new connection to a Redis client.
func New(ctx context.Context, opt Options) *Conn {
	clientOptions := redis.Options(opt)
	return &Conn{
		connInfo{opt.Addr},
		redis.NewClient(&clientOptions),
	}
}

// Do creates and processes a custom Cmd on the underlying Redis client.
func (c *Conn) Do(ctx context.Context, args ...interface{}) (interface{}, error) {
	sp, _ := c.startSpan(ctx, "redis.Do", fmt.Sprintf("%v", args))
	cmd := c.Client.Do(args...)
	trace.SpanComplete(sp, cmd.Err())
	return cmd.Result()
}

// Close closes the connection to the underlying Redis client.
func (c *Conn) Close(ctx context.Context, args ...interface{}) error {
	sp, _ := c.startSpan(ctx, "redis.Close", "")
	cmd := c.Client.Close()
	trace.SpanComplete(sp, cmd)
	return errors.New(cmd.Error())
}

// Ping can be used to test whether a connection is still alive, or measure latency.
func (c *Conn) Ping(ctx context.Context) (string, error) {
	sp, _ := c.startSpan(ctx, "redis.Ping", "")
	cmd := c.Client.Ping()
	trace.SpanComplete(sp, cmd.Err())
	return cmd.Result()
}
