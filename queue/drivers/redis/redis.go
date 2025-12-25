// Package redis provides a Redis-backed queue implementation using Asynq.
package redis

import (
	"github.com/azizndao/glib/queue"
	"github.com/azizndao/glib/queue/internal/asynq"
)

// New creates a new Redis-backed queue using Asynq.
//
// Configuration example:
//
//	config := queue.Config{
//	    Driver: "redis",
//	    Connection: map[string]any{
//	        "addr":     "localhost:6379",
//	        "password": "",
//	        "db":       0,
//	    },
//	}
//
// For Redis Cluster:
//
//	config := queue.Config{
//	    Driver: "redis",
//	    Connection: map[string]any{
//	        "addrs":    []string{"localhost:7000", "localhost:7001"},
//	        "password": "",
//	    },
//	}
func New(config queue.Config) (queue.Queue, error) {
	return asynq.NewQueue(config)
}

// Driver returns a queue driver function for Redis.
// Use this when registering with the queue manager.
//
// Example:
//
//	manager := queue.NewManager()
//	manager.RegisterDriver("redis", redis.Driver())
func Driver() queue.Driver {
	return New
}
