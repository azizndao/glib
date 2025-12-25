package queue

import (
	"context"
	"time"
)

// Job represents a job that can be dispatched to a queue.
// Jobs are units of work that can be executed asynchronously.
//
// Example usage:
//
//	type SendEmailJob struct {
//	    queue.BaseJob
//	    To      string
//	    Subject string
//	    Body    string
//	}
//
//	func (j *SendEmailJob) Handle(ctx context.Context) error {
//	    // Send email logic
//	    return mailer.Send(j.To, j.Subject, j.Body)
//	}
//
//	func (j *SendEmailJob) Failed(ctx context.Context, err error) {
//	    log.Error("Failed to send email", "to", j.To, "error", err)
//	}
type Job interface {
	// Handle executes the job's logic.
	// The context may contain timeout and cancellation information.
	Handle(ctx context.Context) error

	// Queue returns the queue name this job should be pushed to.
	// Return "" to use the default queue.
	Queue() string

	// Tries returns the maximum number of attempts for this job.
	// Return 0 to use the default (1 attempt).
	Tries() int

	// Timeout returns the maximum time this job can run.
	// Return 0 to use the default timeout.
	Timeout() time.Duration

	// RetryDelay returns the delay before retrying a failed job.
	// The attempt parameter starts at 1 for the first retry.
	// Return 0 to use exponential backoff (default behavior).
	RetryDelay(attempt int) time.Duration

	// Failed is called when the job has exhausted all retry attempts.
	// This is useful for logging, notifications, or cleanup.
	Failed(ctx context.Context, err error)
}

// BaseJob provides default implementations for the Job interface.
// Embed this in your job structs to inherit sensible defaults.
//
// Example:
//
//	type MyJob struct {
//	    queue.BaseJob
//	    Data string
//	}
//
//	func (j *MyJob) Handle(ctx context.Context) error {
//	    // Your job logic here
//	    return nil
//	}
type BaseJob struct{}

// Queue returns the default queue name (empty string).
func (b *BaseJob) Queue() string {
	return ""
}

// Tries returns the default number of attempts (1).
func (b *BaseJob) Tries() int {
	return 1
}

// Timeout returns the default timeout (0 = no timeout).
func (b *BaseJob) Timeout() time.Duration {
	return 0
}

// RetryDelay returns the default retry delay (0 = exponential backoff).
func (b *BaseJob) RetryDelay(attempt int) time.Duration {
	return 0
}

// Failed provides a default no-op implementation for job failure handling.
func (b *BaseJob) Failed(ctx context.Context, err error) {
	// Default: do nothing
}

// Handle must be implemented by the embedding struct.
func (b *BaseJob) Handle(ctx context.Context) error {
	panic("queue: Handle method must be implemented")
}

// ShouldQueue is an optional interface that jobs can implement to
// determine whether they should be queued or executed synchronously.
type ShouldQueue interface {
	ShouldQueue() bool
}

// Middleware is an optional interface that jobs can implement to
// specify middleware that should be applied when processing the job.
type Middleware interface {
	Middleware() []JobMiddleware
}

// JobMiddleware is a function that wraps job execution.
// It receives the next handler and returns a new handler.
type JobMiddleware func(next JobHandler) JobHandler

// JobHandler is a function that executes a job.
type JobHandler func(ctx context.Context, job Job) error

// UniqueJob is an optional interface that jobs can implement to
// prevent duplicate jobs from being queued.
type UniqueJob interface {
	// UniqueID returns a unique identifier for the job.
	// Jobs with the same UniqueID will not be queued if one is already pending.
	UniqueID() string

	// UniqueTTL returns how long the uniqueness lock should be held.
	// Return 0 to hold the lock until the job completes.
	UniqueTTL() time.Duration
}

// Releasable is an optional interface that jobs can implement to
// control job release behavior when a job fails.
type Releasable interface {
	// Release returns whether the job should be released back to the queue
	// instead of being retried. The delay parameter specifies how long to wait
	// before making the job available again.
	Release() (shouldRelease bool, delay time.Duration)
}

// Deletable is an optional interface that jobs can implement to
// determine if a job should be deleted without processing.
type Deletable interface {
	// ShouldDelete returns true if the job should be deleted without processing.
	// This is useful for conditional job execution.
	ShouldDelete() bool
}
