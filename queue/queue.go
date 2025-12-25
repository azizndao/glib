// Package queue provides a flexible job queue system with support for
// multiple drivers (Redis via Asynq, database), delayed jobs, retries,
// unique jobs, and job chaining.
//
// Example usage:
//
//	// Create queue manager
//	manager := queue.NewManager()
//	manager.RegisterDriver("redis", func(config Config) (Queue, error) {
//	    return redis.New(config)
//	})
//
//	// Get default queue
//	q, _ := manager.Default()
//
//	// Dispatch a job
//	queue.Dispatch(&SendEmailJob{
//	    To:      "user@example.com",
//	    Subject: "Welcome",
//	    Body:    "Welcome to our service!",
//	}).Dispatch()
//
//	// Dispatch with options
//	queue.Dispatch(&ProcessVideoJob{VideoID: 1}).
//	    OnQueue("videos").
//	    Delay(5 * time.Minute).
//	    Dispatch()
//
//	// Start a worker
//	worker := queue.NewWorker(manager)
//	worker.Work(context.Background())
package queue

import (
	"context"
)

// Queue is the interface that all queue drivers must implement.
type Queue interface {
	// Push pushes a job to the queue with the given options.
	Push(ctx context.Context, job Job, opts *Options) (string, error)

	// Pop retrieves and removes the next job from the specified queues.
	// It blocks until a job is available or the context is cancelled.
	Pop(ctx context.Context, queues []string) (Job, error)

	// Delete removes a job from the queue by ID.
	Delete(ctx context.Context, id string) error

	// Release releases a job back to the queue to be processed again.
	// The job will be available after the specified delay.
	Release(ctx context.Context, id string, opts *Options) error

	// Info returns information about a specific job.
	Info(ctx context.Context, id string) (*JobInfo, error)

	// Stats returns statistics about the specified queues.
	// If no queues are specified, returns stats for all queues.
	Stats(ctx context.Context, queues ...string) ([]*QueueStats, error)

	// Clear removes all jobs from the specified queue.
	Clear(ctx context.Context, queue string) error

	// Pause pauses processing of the specified queue.
	Pause(ctx context.Context, queue string) error

	// Resume resumes processing of the specified queue.
	Resume(ctx context.Context, queue string) error

	// Close closes the queue connection and releases resources.
	Close() error
}

// Driver is a function that creates a queue instance from configuration.
type Driver func(config Config) (Queue, error)

// Inspector provides methods for inspecting and managing jobs in the queue.
type Inspector interface {
	// ListQueues returns a list of all queue names.
	ListQueues(ctx context.Context) ([]string, error)

	// ListPending returns pending jobs in the specified queue.
	ListPending(ctx context.Context, queue string, page, pageSize int) ([]*JobInfo, error)

	// ListActive returns active jobs in the specified queue.
	ListActive(ctx context.Context, queue string, page, pageSize int) ([]*JobInfo, error)

	// ListScheduled returns scheduled jobs in the specified queue.
	ListScheduled(ctx context.Context, queue string, page, pageSize int) ([]*JobInfo, error)

	// ListRetry returns retry jobs in the specified queue.
	ListRetry(ctx context.Context, queue string, page, pageSize int) ([]*JobInfo, error)

	// ListFailed returns failed jobs in the specified queue.
	ListFailed(ctx context.Context, queue string, page, pageSize int) ([]*JobInfo, error)

	// RetryJob retries a failed job.
	RetryJob(ctx context.Context, id string) error

	// RetryAllFailed retries all failed jobs in the specified queue.
	RetryAllFailed(ctx context.Context, queue string) (int, error)

	// DeleteJob deletes a job by ID.
	DeleteJob(ctx context.Context, id string) error

	// DeleteAllFailed deletes all failed jobs in the specified queue.
	DeleteAllFailed(ctx context.Context, queue string) (int, error)

	// ArchiveJob archives a job by ID.
	ArchiveJob(ctx context.Context, id string) error

	// ArchiveAllFailed archives all failed jobs in the specified queue.
	ArchiveAllFailed(ctx context.Context, queue string) (int, error)
}

// Worker processes jobs from queues.
type Worker interface {
	// Start starts the worker to process jobs.
	Start(ctx context.Context) error

	// Stop stops the worker gracefully.
	Stop() error

	// RegisterHandler registers a job handler.
	RegisterHandler(jobType string, handler JobHandler)

	// RegisterMiddleware registers middleware that will be applied to all jobs.
	RegisterMiddleware(middleware ...JobMiddleware)
}
