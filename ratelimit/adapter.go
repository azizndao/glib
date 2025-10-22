package ratelimit

import (
	"context"
)

// RedisClientAdapter adapts go-redis client to RedisCommander interface
// This allows seamless integration with github.com/redis/go-redis/v9
//
// Example usage:
//
//	import (
//	    "github.com/redis/go-redis/v9"
//	    "github.com/azizndao/glib/ratelimit"
//	)
//
//	redisClient := redis.NewClient(&redis.Options{
//	    Addr: "localhost:6379",
//	})
//
//	adapter := ratelimit.NewRedisClientAdapter(redisClient)
//	store := ratelimit.NewRedisStore(adapter, "ratelimit:")
//
//	router.Use(ratelimit.RateLimit(ratelimit.Config{
//	    Max:    100,
//	    Window: time.Minute,
//	    Store:  store,
//	}))

// GoRedisClient is the interface that go-redis clients implement
// This matches both redis.Client and redis.ClusterClient
type GoRedisClient interface {
	Eval(ctx context.Context, script string, keys []string, args ...any) GoRedisCmd
	Get(ctx context.Context, key string) GoRedisStringCmd
	Del(ctx context.Context, keys ...string) GoRedisIntCmd
}

// GoRedisCmd represents go-redis Cmd
type GoRedisCmd interface {
	Result() (any, error)
}

// GoRedisStringCmd represents go-redis StringCmd
type GoRedisStringCmd interface {
	Result() (string, error)
}

// GoRedisIntCmd represents go-redis IntCmd
type GoRedisIntCmd interface {
	Result() (int64, error)
}

// RedisClientAdapter wraps a go-redis client to implement RedisCommander
type RedisClientAdapter struct {
	client GoRedisClient
}

// NewRedisClientAdapter creates an adapter for go-redis client
func NewRedisClientAdapter(client GoRedisClient) *RedisClientAdapter {
	return &RedisClientAdapter{client: client}
}

// Eval executes a Lua script
func (a *RedisClientAdapter) Eval(ctx context.Context, script string, keys []string, args ...any) (any, error) {
	return a.client.Eval(ctx, script, keys, args...).Result()
}

// Get gets the value of a key
func (a *RedisClientAdapter) Get(ctx context.Context, key string) (string, error) {
	result, err := a.client.Get(ctx, key).Result()
	if err != nil {
		// Check if it's a "key not found" error from go-redis
		// go-redis returns redis.Nil for missing keys
		// We check the error message since we don't import redis package
		if err.Error() == "redis: nil" {
			return "", nil
		}
		return "", err
	}
	return result, nil
}

// Del deletes one or more keys
func (a *RedisClientAdapter) Del(ctx context.Context, keys ...string) (int64, error) {
	return a.client.Del(ctx, keys...).Result()
}
