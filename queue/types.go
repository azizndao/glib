package queue

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrJobNotFound is returned when a job cannot be found
	ErrJobNotFound = errors.New("queue: job not found")

	// ErrInvalidJob is returned when a job is invalid
	ErrInvalidJob = errors.New("queue: invalid job")

	// ErrQueueNotFound is returned when a queue connection is not found
	ErrQueueNotFound = errors.New("queue: queue connection not found")

	// ErrDriverNotRegistered is returned when a queue driver is not registered
	ErrDriverNotRegistered = errors.New("queue: driver not registered")

	// ErrJobSerializationFailed is returned when job serialization fails
	ErrJobSerializationFailed = errors.New("queue: job serialization failed")

	// ErrJobDeserializationFailed is returned when job deserialization fails
	ErrJobDeserializationFailed = errors.New("queue: job deserialization failed")

	// ErrDuplicateJob is returned when attempting to queue a duplicate unique job
	ErrDuplicateJob = errors.New("queue: duplicate job")
)

// Options contains options for dispatching a job.
type Options struct {
	// Queue is the name of the queue to push the job to
	Queue string

	// Connection is the name of the queue connection to use
	Connection string

	// Delay specifies how long to wait before making the job available
	Delay time.Duration

	// MaxRetries is the maximum number of retry attempts
	MaxRetries int

	// Timeout is the maximum time the job can run
	Timeout time.Duration

	// Deadline is the absolute time by which the job must complete
	Deadline time.Time

	// Retention is how long to keep the job in the completed state
	Retention time.Duration

	// UniqueFor is how long the job should remain unique
	UniqueFor time.Duration

	// TaskID is a unique identifier for the job (generated if not provided)
	TaskID string

	// Group is used for grouping related jobs
	Group string
}

// DefaultOptions returns options with default values.
func DefaultOptions() *Options {
	return &Options{
		Queue:      "default",
		Connection: "default",
		MaxRetries: 1,
		TaskID:     uuid.New().String(),
	}
}

// JobInfo contains information about a queued job.
type JobInfo struct {
	// ID is the unique identifier for the job
	ID string

	// Type is the job type name
	Type string

	// Queue is the queue name
	Queue string

	// State is the current state of the job
	State JobState

	// MaxRetries is the maximum number of retry attempts
	MaxRetries int

	// Retried is the number of times the job has been retried
	Retried int

	// NextProcessAt is when the job will be processed next
	NextProcessAt time.Time

	// LastError is the last error message if the job failed
	LastError string

	// CompletedAt is when the job completed (if completed)
	CompletedAt *time.Time
}

// JobState represents the state of a job in the queue.
type JobState string

const (
	// JobStatePending indicates the job is waiting to be processed
	JobStatePending JobState = "pending"

	// JobStateActive indicates the job is currently being processed
	JobStateActive JobState = "active"

	// JobStateCompleted indicates the job completed successfully
	JobStateCompleted JobState = "completed"

	// JobStateFailed indicates the job failed and will not be retried
	JobStateFailed JobState = "failed"

	// JobStateScheduled indicates the job is scheduled for future execution
	JobStateScheduled JobState = "scheduled"

	// JobStateRetry indicates the job is scheduled for retry
	JobStateRetry JobState = "retry"

	// JobStateArchived indicates the job has been archived
	JobStateArchived JobState = "archived"
)

// QueueStats contains statistics about a queue.
type QueueStats struct {
	// Queue is the queue name
	Queue string

	// Pending is the number of pending jobs
	Pending int

	// Active is the number of active jobs
	Active int

	// Scheduled is the number of scheduled jobs
	Scheduled int

	// Retry is the number of jobs waiting for retry
	Retry int

	// Failed is the number of failed jobs
	Failed int

	// Completed is the number of completed jobs
	Completed int

	// Processed is the total number of jobs processed
	Processed int

	// ProcessedToday is the number of jobs processed today
	ProcessedToday int
}

// Config contains configuration for a queue connection.
type Config struct {
	// Driver is the queue driver name (e.g., "redis", "database")
	Driver string

	// Connection is a map of driver-specific configuration
	Connection map[string]any

	// Queue is the default queue name
	Queue string

	// RetryAfter is the default time after which a job is considered failed
	// if not completed (for database driver)
	RetryAfter time.Duration

	// BlockFor is how long to wait for a job when using long polling
	BlockFor time.Duration
}
