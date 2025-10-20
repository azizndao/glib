package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// RedisCommander is the minimal interface needed for rate limiting with Redis
// Compatible with both redis.Client and redis.ClusterClient from go-redis
type RedisCommander interface {
	// Eval executes a Lua script
	Eval(ctx context.Context, script string, keys []string, args ...any) (any, error)
	// Get gets the value of a key
	Get(ctx context.Context, key string) (string, error)
	// Del deletes one or more keys
	Del(ctx context.Context, keys ...string) (int64, error)
}

// RedisStore implements Store interface using Redis
type RedisStore struct {
	client RedisCommander
	prefix string
}

// NewRedisStore creates a new Redis-based store for rate limiting
// client: A RedisCommander implementation (you can wrap go-redis client with RedisClientAdapter)
// prefix: Key prefix for rate limit entries (e.g., "ratelimit:")
//
// Example with go-redis:
//
//	import "github.com/redis/go-redis/v9"
//
//	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//	adapter := ratelimit.NewRedisClientAdapter(redisClient)
//	store := ratelimit.NewRedisStore(adapter, "ratelimit:")
func NewRedisStore(client RedisCommander, prefix string) *RedisStore {
	if prefix == "" {
		prefix = "ratelimit:"
	}
	return &RedisStore{
		client: client,
		prefix: prefix,
	}
}

// Lua script for atomic increment with sliding window
// This ensures atomicity and accuracy even under high concurrency
const incrementScript = `
local key = KEYS[1]
local window = tonumber(ARGV[1])
local now = tonumber(ARGV[2])

-- Get current count and window start
local data = redis.call('HMGET', key, 'count', 'window_start')
local count = tonumber(data[1]) or 0
local window_start = tonumber(data[2]) or now

-- Check if window has expired
if now - window_start > window then
    -- Reset window
    count = 1
    window_start = now
else
    -- Increment count
    count = count + 1
end

-- Update Redis
redis.call('HMSET', key, 'count', count, 'window_start', window_start)
redis.call('EXPIRE', key, window * 2)

-- Calculate TTL (time until window expires)
local ttl = window - (now - window_start)

return {count, ttl}
`

// Increment increments the counter for the given key using Lua script for atomicity
func (r *RedisStore) Increment(ctx context.Context, key string, window time.Duration) (int, time.Duration, error) {
	fullKey := r.prefix + key
	windowSeconds := int64(window.Seconds())
	now := time.Now().Unix()

	// Execute Lua script
	result, err := r.client.Eval(ctx, incrementScript, []string{fullKey}, windowSeconds, now)
	if err != nil {
		return 0, 0, fmt.Errorf("redis increment failed: %w", err)
	}

	// Parse result
	arr, ok := result.([]any)
	if !ok || len(arr) != 2 {
		return 0, 0, fmt.Errorf("unexpected redis response format")
	}

	count, err := toInt(arr[0])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse count: %w", err)
	}

	ttlSeconds, err := toInt(arr[1])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse ttl: %w", err)
	}

	ttl := time.Duration(ttlSeconds) * time.Second
	return count, ttl, nil
}

// Get returns the current count for the given key
func (r *RedisStore) Get(ctx context.Context, key string) (int, time.Duration, error) {
	fullKey := r.prefix + key

	val, err := r.client.Get(ctx, fullKey)
	if err != nil {
		// Key doesn't exist
		return 0, 0, nil
	}

	count, err := strconv.Atoi(val)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse count: %w", err)
	}

	return count, 0, nil
}

// Reset resets the counter for the given key
func (r *RedisStore) Reset(ctx context.Context, key string) error {
	fullKey := r.prefix + key
	_, err := r.client.Del(ctx, fullKey)
	return err
}

// Close closes the Redis store (no-op for Redis as connection is managed externally)
func (r *RedisStore) Close() error {
	// Redis client lifecycle is managed by the caller
	return nil
}

// toInt converts interface{} to int (handles both int64 and string from Redis)
func toInt(val interface{}) (int, error) {
	switch v := val.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("cannot convert %T to int", val)
	}
}
