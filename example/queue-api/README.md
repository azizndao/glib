# Queue API Example

This example demonstrates how to use the Glib queue system to dispatch and process background jobs.

## Features Demonstrated

- **Multiple Job Types**: Email, video processing, and report generation
- **Queue Priority**: Different queues with priority levels
- **Delayed Jobs**: Jobs that execute after a delay
- **Retry Logic**: Automatic retry with configurable attempts
- **Worker Management**: Concurrent job processing
- **Queue Statistics**: Real-time queue metrics

## Prerequisites

- Go 1.21 or later
- Redis server running on localhost:6379 (or configure via environment variables)

```bash
# Install dependencies
go mod tidy

# Or manually
go get github.com/azizndao/glib/queue
go get github.com/azizndao/glib/queue/drivers/redis
```

## Running the Example

### Start Redis

```bash
# Using Docker
docker run -d -p 6379:6379 redis:latest

# Or install Redis locally
```

### Run the API Server

```bash
go run main.go
# Or with environment variables
REDIS_ADDR=localhost:6379 go run main.go
```

The server will start on http://localhost:8080

### Run the Worker

In a separate terminal:

```bash
MODE=worker go run main.go
```

## API Endpoints

### Dispatch Jobs

```bash
# Send an email job
curl -X POST "http://localhost:8080/send-email?to=user@example.com&subject=Hello&body=Test"

# Process a video (with 5-second delay)
curl -X POST "http://localhost:8080/process-video"

# Generate a report
curl -X POST "http://localhost:8080/generate-report"
```

### Get Queue Statistics

```bash
curl http://localhost:8080/stats
```

## Job Examples

### SendEmailJob

```go
type SendEmailJob struct {
    queue.BaseJob
    To      string
    Subject string
    Body    string
}

func (j *SendEmailJob) Handle(ctx context.Context) error {
    // Send email logic
    return mailer.Send(j.To, j.Subject, j.Body)
}

func (j *SendEmailJob) Queue() string {
    return "emails"  // Use the emails queue
}

func (j *SendEmailJob) Tries() int {
    return 3  // Retry up to 3 times
}
```

### Dispatching Jobs

```go
// Simple dispatch
queue.Dispatch(&SendEmailJob{
    To:      "user@example.com",
    Subject: "Welcome",
    Body:    "Welcome to our service!",
}).Dispatch()

// With options
queue.Dispatch(&ProcessVideoJob{VideoID: 1}).
    OnQueue("videos").
    Delay(5 * time.Minute).
    MaxRetries(3).
    Dispatch()
```

### Worker Configuration

```go
config := queue.WorkerConfig{
    Concurrency: 5,
    Queues: map[string]int{
        "emails":  3, // Higher priority
        "videos":  2,
        "default": 1, // Lower priority
    },
    StrictPriority: false,
}

worker := queue.NewWorker(manager, config)
worker.RegisterJobs(
    &SendEmailJob{},
    &ProcessVideoJob{},
    &GenerateReportJob{},
)
worker.Work(context.Background())
```

## Architecture

```
┌─────────────┐      ┌─────────┐      ┌─────────────┐
│   Client    │─────▶│  Redis  │◀─────│   Worker    │
│   (API)     │      │ (Asynq) │      │  (Process)  │
└─────────────┘      └─────────┘      └─────────────┘
      │                    │                   │
      │  Dispatch Jobs     │   Poll Jobs      │
      └────────────────────┴──────────────────┘
```

## Environment Variables

- `MODE`: Run mode (`server` or `worker`) - default: `server`
- `REDIS_ADDR`: Redis address - default: `localhost:6379`
- `REDIS_PASSWORD`: Redis password - default: `` (empty)

## Monitoring

You can monitor the queue using:

1. **API Statistics Endpoint**: `GET /stats`
2. **Asynq Web UI**: (Optional) Run `asynqmon` for a web dashboard
3. **Logs**: The worker logs all job processing events

## Next Steps

- Add job chaining (execute jobs sequentially)
- Implement job batching (group related jobs)
- Add unique jobs (prevent duplicates)
- Integrate with the Glib HTTP server for full-stack applications
