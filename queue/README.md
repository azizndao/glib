# Glib Queue

A powerful and flexible job queue system for Go, inspired by Laravel's queue system. Built on top of [Asynq](https://github.com/hibiken/asynq), it provides a clean, Laravel-like API for dispatching and processing background jobs.

## Features

- **Laravel-style API**: Familiar, fluent interface for dispatching jobs
- **Multiple Queue Support**: Prioritize jobs across different queues
- **Delayed Jobs**: Schedule jobs to run at a specific time
- **Job Retries**: Automatic retry with exponential backoff
- **Job Chaining**: Execute jobs sequentially
- **Job Batching**: Group related jobs together
- **Unique Jobs**: Prevent duplicate jobs from being queued
- **Event System**: Hook into job lifecycle events
- **Graceful Shutdown**: Workers stop gracefully on termination signals
- **Redis-backed**: Reliable job storage using Redis via Asynq
- **Extensible**: Easy to add custom drivers

## Installation

```bash
go get github.com/azizndao/glib/queue
```

## Quick Start

### 1. Define a Job

```go
package jobs

import (
    "context"
    "log/slog"
    
    "github.com/azizndao/glib/queue"
)

type SendEmailJob struct {
    queue.BaseJob
    To      string
    Subject string
    Body    string
}

func (j *SendEmailJob) Handle(ctx context.Context) error {
    slog.Info("Sending email", "to", j.To)
    // Your email sending logic here
    return mailer.Send(j.To, j.Subject, j.Body)
}

func (j *SendEmailJob) Queue() string {
    return "emails"  // Queue name
}

func (j *SendEmailJob) Tries() int {
    return 3  // Retry up to 3 times
}

func (j *SendEmailJob) Failed(ctx context.Context, err error) {
    slog.Error("Failed to send email", "to", j.To, "error", err)
}
```

### 2. Set Up the Queue Manager

```go
package main

import (
    "github.com/azizndao/glib/queue"
    "github.com/azizndao/glib/queue/drivers/redis"
)

func main() {
    // Create queue manager
    manager := queue.NewManager()
    
    // Register Redis driver (Asynq)
    manager.RegisterDriver("redis", redis.New)
    
    // Register connection configuration
    manager.RegisterConfig("default", queue.Config{
        Driver: "redis",
        Connection: map[string]any{
            "addr":     "localhost:6379",
            "password": "",
            "db":       0,
        },
    })
    
    // Set as global default
    queue.SetDefaultManager(manager)
    
    // Register job types
    queue.Register(&SendEmailJob{})
}
```

### 3. Dispatch Jobs

```go
// Simple dispatch
queue.Dispatch(&SendEmailJob{
    To:      "user@example.com",
    Subject: "Welcome",
    Body:    "Welcome to our service!",
}).Dispatch()

// With options
queue.Dispatch(&SendEmailJob{
    To:      "user@example.com",
    Subject: "Welcome",
    Body:    "Welcome!",
}).
    OnQueue("high-priority").
    Delay(5 * time.Minute).
    MaxRetries(5).
    Dispatch()
```

### 4. Start a Worker

```go
// Create worker
worker := queue.NewWorker(manager, queue.WorkerConfig{
    Concurrency: 10,
    Queues: map[string]int{
        "high-priority": 6,
        "emails":        3,
        "default":       1,
    },
})

// Register jobs
worker.RegisterJobs(
    &SendEmailJob{},
    &ProcessVideoJob{},
)

// Start processing
ctx := context.Background()
worker.Work(ctx)
```

## Job Interface

All jobs must implement the `Job` interface:

```go
type Job interface {
    Handle(ctx context.Context) error
    Queue() string
    Tries() int
    Timeout() time.Duration
    RetryDelay(attempt int) time.Duration
    Failed(ctx context.Context, err error)
}
```

### BaseJob

Embed `queue.BaseJob` to get sensible defaults:

```go
type MyJob struct {
    queue.BaseJob
    Data string
}

func (j *MyJob) Handle(ctx context.Context) error {
    // Your logic here
    return nil
}
```

The `BaseJob` provides:
- Default queue name: `""` (uses default queue)
- Default tries: `1`
- Default timeout: `0` (no timeout)
- Default retry delay: `0` (exponential backoff)
- No-op `Failed` handler

## Dispatching Jobs

### Fluent API

```go
queue.Dispatch(&MyJob{Data: "test"}).
    OnQueue("processing").              // Specify queue
    OnConnection("redis").               // Specify connection
    Delay(10 * time.Minute).            // Delay execution
    DelayUntil(time.Now().Add(1*time.Hour)). // Delay until time
    MaxRetries(3).                       // Override max retries
    Timeout(30 * time.Second).          // Set timeout
    WithTaskID("unique-id").            // Custom task ID
    InGroup("batch-1").                 // Group jobs
    Dispatch()                           // Execute dispatch
```

### Conditional Dispatch

```go
// Dispatch only if condition is true
queue.Dispatch(&MyJob{}).DispatchIf(user.WantsEmails)

// Dispatch unless condition is true
queue.Dispatch(&MyJob{}).DispatchUnless(user.Unsubscribed)
```

### Synchronous Execution

For testing or immediate execution:

```go
err := queue.Dispatch(&MyJob{}).DispatchSync()
```

## Job Chaining

Execute jobs sequentially, where each job only runs if the previous succeeded:

```go
queue.NewChain(
    &ProcessVideo{VideoID: 1},
    &OptimizeVideo{VideoID: 1},
    &NotifyUser{UserID: 100},
).
    OnQueue("videos").
    Dispatch()
```

## Job Batching

Dispatch multiple jobs that can run in parallel:

```go
jobs := []queue.Job{
    &SendEmailJob{To: "user1@example.com"},
    &SendEmailJob{To: "user2@example.com"},
    &SendEmailJob{To: "user3@example.com"},
}

queue.NewBatch(jobs...).
    OnQueue("emails").
    Dispatch()
```

## Unique Jobs

Prevent duplicate jobs from being queued:

```go
type UniqueProcessJob struct {
    queue.BaseJob
    UserID int
}

func (j *UniqueProcessJob) UniqueID() string {
    return fmt.Sprintf("process-user-%d", j.UserID)
}

func (j *UniqueProcessJob) UniqueTTL() time.Duration {
    return 1 * time.Hour  // Unique for 1 hour
}
```

## Worker Configuration

```go
config := queue.WorkerConfig{
    // Connection name to use
    Connection: "default",
    
    // Maximum number of concurrent jobs
    Concurrency: 10,
    
    // Queue priorities (higher = more important)
    Queues: map[string]int{
        "critical": 6,
        "high":     4,
        "default":  2,
        "low":      1,
    },
    
    // Strictly prioritize higher queues
    StrictPriority: false,
    
    // Graceful shutdown timeout
    ShutdownTimeout: 20 * time.Second,
    
    // Health check interval
    HealthCheckInterval: 15 * time.Second,
    
    // Custom logger
    Logger: slog.Default(),
}

worker := queue.NewWorker(manager, config)
```

### Priority Modes

- **Weighted Priority** (`StrictPriority: false`): Workers process jobs proportionally based on weights
- **Strict Priority** (`StrictPriority: true`): Workers only process lower priority queues when higher ones are empty

## Advanced Job Features

### Custom Retry Delay

```go
func (j *MyJob) RetryDelay(attempt int) time.Duration {
    // Linear backoff: 1s, 2s, 3s, etc.
    return time.Duration(attempt) * time.Second
    
    // Or exponential: 1s, 2s, 4s, 8s, etc.
    return time.Duration(math.Pow(2, float64(attempt))) * time.Second
}
```

### Job Timeout

```go
func (j *LongRunningJob) Timeout() time.Duration {
    return 5 * time.Minute
}
```

### Conditional Deletion

```go
type ConditionalJob struct {
    queue.BaseJob
    ShouldProcess bool
}

func (j *ConditionalJob) ShouldDelete() bool {
    return !j.ShouldProcess
}
```

### Release Instead of Retry

```go
func (j *MyJob) Release() (shouldRelease bool, delay time.Duration) {
    // Release the job back to queue instead of retrying
    return true, 30 * time.Second
}
```

## Job Middleware

Apply middleware to all jobs processed by a worker:

```go
// Logging middleware
loggingMiddleware := func(next queue.JobHandler) queue.JobHandler {
    return func(ctx context.Context, job queue.Job) error {
        start := time.Now()
        err := next(ctx, job)
        duration := time.Since(start)
        
        slog.Info("Job processed",
            "job", queue.TypeName(job),
            "duration", duration,
            "error", err,
        )
        
        return err
    }
}

worker.RegisterMiddleware(loggingMiddleware)
```

## Events

Listen to job lifecycle events:

```go
// Job dispatched
queue.Listen("job.dispatched", func(ctx context.Context, event queue.Event) error {
    e := event.(*queue.JobDispatchedEvent)
    log.Printf("Job %s dispatched to queue %s", e.JobID, e.Queue)
    return nil
})

// Job processing
queue.Listen("job.processing", func(ctx context.Context, event queue.Event) error {
    e := event.(*queue.JobProcessingEvent)
    log.Printf("Processing job %s", e.JobID)
    return nil
})

// Job processed successfully
queue.Listen("job.processed", func(ctx context.Context, event queue.Event) error {
    e := event.(*queue.JobProcessedEvent)
    log.Printf("Job %s processed successfully", e.JobID)
    return nil
})

// Job failed
queue.Listen("job.failed", func(ctx context.Context, event queue.Event) error {
    e := event.(*queue.JobFailedEvent)
    log.Printf("Job %s failed: %v (attempt %d)", e.JobID, e.Error, e.Attempt)
    return nil
})
```

## Queue Statistics

Get real-time statistics about your queues:

```go
q, _ := manager.Default()
stats, _ := q.Stats(ctx)

for _, stat := range stats {
    fmt.Printf("Queue: %s\n", stat.Queue)
    fmt.Printf("  Pending: %d\n", stat.Pending)
    fmt.Printf("  Active: %d\n", stat.Active)
    fmt.Printf("  Scheduled: %d\n", stat.Scheduled)
    fmt.Printf("  Failed: %d\n", stat.Failed)
    fmt.Printf("  Processed: %d\n", stat.Processed)
}
```

## Testing

### Unit Testing Jobs

```go
func TestSendEmailJob(t *testing.T) {
    job := &SendEmailJob{
        To:      "test@example.com",
        Subject: "Test",
        Body:    "Test email",
    }
    
    err := job.Handle(context.Background())
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }
}
```

### Integration Testing with Test Queue

For integration tests, you can use a test Redis instance or mock the queue interface.

## Production Deployment

### Running Workers

```bash
# Single worker
./myapp worker

# Multiple workers (for horizontal scaling)
./myapp worker --concurrency=20 --queues=high:6,default:3,low:1
```

### Monitoring

1. **Asynq Web UI**: Run `asynqmon` for a web-based dashboard
2. **Logs**: Workers log all job processing events
3. **Metrics**: Integrate with Prometheus using Asynq's metrics

### Docker Example

```dockerfile
FROM golang:1.21-alpine

WORKDIR /app
COPY . .
RUN go build -o worker .

CMD ["./worker"]
```

```yaml
# docker-compose.yml
version: '3.8'
services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
  
  worker:
    build: .
    depends_on:
      - redis
    environment:
      - REDIS_ADDR=redis:6379
    deploy:
      replicas: 3
```

## Best Practices

1. **Keep Jobs Small**: Jobs should do one thing well
2. **Make Jobs Idempotent**: Jobs should be safe to run multiple times
3. **Use Unique Jobs**: Prevent duplicate work with unique job IDs
4. **Monitor Queue Depth**: Alert on growing queue backlogs
5. **Set Appropriate Timeouts**: Prevent jobs from running indefinitely
6. **Handle Failures Gracefully**: Log errors and use the `Failed()` callback
7. **Scale Workers Horizontally**: Run multiple worker processes
8. **Use Queue Priorities**: Separate time-sensitive jobs from background tasks

## Comparison with Other Queue Systems

| Feature | Glib Queue | Laravel Queue | Machinery | River |
|---------|-----------|---------------|-----------|-------|
| Laravel-style API | ✅ | ✅ | ❌ | ❌ |
| Redis Backend | ✅ | ✅ | ✅ | ❌ |
| PostgreSQL Backend | 🚧 | ✅ | ✅ | ✅ |
| Job Chaining | ✅ | ✅ | ✅ | ✅ |
| Unique Jobs | ✅ | ✅ | ❌ | ✅ |
| Web UI | ✅ (Asynq) | ✅ (Horizon) | ❌ | 🚧 |
| Production Ready | ✅ | ✅ | ✅ | 🚧 |

## Troubleshooting

### Jobs Not Processing

1. Check Redis connection: `redis-cli ping`
2. Verify worker is running
3. Check queue names match between dispatcher and worker
4. Review worker logs for errors

### High Memory Usage

1. Reduce worker concurrency
2. Check for memory leaks in job handlers
3. Monitor job payload sizes

### Slow Job Processing

1. Increase worker concurrency
2. Scale workers horizontally
3. Optimize job handlers
4. Use queue priorities

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](../CONTRIBUTING.md) for details.

## License

MIT License - see [LICENSE](../LICENSE) for details.

## Credits

Built on top of:
- [Asynq](https://github.com/hibiken/asynq) - Robust distributed task queue
- Inspired by [Laravel Queue](https://laravel.com/docs/queues)
