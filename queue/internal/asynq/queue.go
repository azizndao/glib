package asynq

import (
	"context"
	"fmt"
	"time"

	"github.com/azizndao/glib/queue"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// Queue implements the queue.Queue interface using Asynq.
type Queue struct {
	client     *asynq.Client
	inspector  *asynq.Inspector
	serializer *queue.Serializer
	registry   *queue.JobRegistry
}

// New creates a new Asynq queue implementation.
func New(config queue.Config) (*Queue, error) {
	// Extract Redis configuration
	redisOpt := parseRedisConfig(config.Connection)

	// Create Asynq client
	client := asynq.NewClient(redisOpt)

	// Create Asynq inspector
	inspector := asynq.NewInspector(redisOpt)

	// Create or use existing registry
	registry := queue.NewJobRegistry()

	// Create serializer (use gob by default)
	serializer := queue.NewSerializer(registry, "gob")

	return &Queue{
		client:     client,
		inspector:  inspector,
		serializer: serializer,
		registry:   registry,
	}, nil
}

// Push pushes a job to the queue.
func (q *Queue) Push(ctx context.Context, job queue.Job, opts *queue.Options) (string, error) {
	if opts == nil {
		opts = queue.DefaultOptions()
	}

	// Serialize the job
	payload, err := q.serializer.Serialize(job)
	if err != nil {
		return "", err
	}

	// Create Asynq task
	task := asynq.NewTask(q.registry.TypeName(job), payload)

	// Build Asynq options
	asynqOpts := q.buildTaskOptions(job, opts)

	// Enqueue the task
	info, err := q.client.EnqueueContext(ctx, task, asynqOpts...)
	if err != nil {
		return "", fmt.Errorf("failed to enqueue task: %w", err)
	}

	return info.ID, nil
}

// Pop is not directly supported by Asynq (use Worker instead).
func (q *Queue) Pop(ctx context.Context, queues []string) (queue.Job, error) {
	return nil, fmt.Errorf("Pop is not supported with Asynq driver, use Worker instead")
}

// Delete removes a job from the queue.
func (q *Queue) Delete(ctx context.Context, id string) error {
	// Try deleting from all possible queues - Asynq requires queue name
	queues, err := q.inspector.Queues()
	if err != nil {
		return fmt.Errorf("failed to list queues: %w", err)
	}

	for _, queueName := range queues {
		if err := q.inspector.DeleteTask(queueName, id); err == nil {
			return nil // Successfully deleted
		}
	}

	return queue.ErrJobNotFound
}

// Release is not directly supported by Asynq.
func (q *Queue) Release(ctx context.Context, id string, opts *queue.Options) error {
	return fmt.Errorf("Release is not supported with Asynq driver")
}

// Info returns information about a job.
func (q *Queue) Info(ctx context.Context, id string) (*queue.JobInfo, error) {
	// Try to find the task in different queues
	queues, err := q.inspector.Queues()
	if err != nil {
		return nil, fmt.Errorf("failed to list queues: %w", err)
	}

	for _, queueName := range queues {
		taskInfo, err := q.inspector.GetTaskInfo(queueName, id)
		if err == nil {
			return q.convertTaskInfo(taskInfo), nil
		}
	}

	return nil, queue.ErrJobNotFound
}

// Stats returns statistics about queues.
func (q *Queue) Stats(ctx context.Context, queues ...string) ([]*queue.QueueStats, error) {
	if len(queues) == 0 {
		// Get all queues
		queueNames, err := q.inspector.Queues()
		if err != nil {
			return nil, fmt.Errorf("failed to list queues: %w", err)
		}
		queues = queueNames
	}

	stats := make([]*queue.QueueStats, 0, len(queues))
	for _, queueName := range queues {
		info, err := q.inspector.GetQueueInfo(queueName)
		if err != nil {
			continue // Skip queues that don't exist
		}

		stats = append(stats, &queue.QueueStats{
			Queue:     queueName,
			Pending:   info.Pending,
			Active:    info.Active,
			Scheduled: info.Scheduled,
			Retry:     info.Retry,
			Failed:    info.Archived, // Asynq calls failed jobs "archived"
			Completed: info.Completed,
			Processed: info.Processed,
		})
	}

	return stats, nil
}

// Clear removes all jobs from a queue.
func (q *Queue) Clear(ctx context.Context, queueName string) error {
	// Delete all pending tasks
	if _, err := q.inspector.DeleteAllPendingTasks(queueName); err != nil {
		return fmt.Errorf("failed to clear pending tasks: %w", err)
	}

	// Delete all scheduled tasks
	if _, err := q.inspector.DeleteAllScheduledTasks(queueName); err != nil {
		return fmt.Errorf("failed to clear scheduled tasks: %w", err)
	}

	// Delete all retry tasks
	if _, err := q.inspector.DeleteAllRetryTasks(queueName); err != nil {
		return fmt.Errorf("failed to clear retry tasks: %w", err)
	}

	return nil
}

// Pause pauses a queue.
func (q *Queue) Pause(ctx context.Context, queueName string) error {
	return q.inspector.PauseQueue(queueName)
}

// Resume resumes a queue.
func (q *Queue) Resume(ctx context.Context, queueName string) error {
	return q.inspector.UnpauseQueue(queueName)
}

// Close closes the queue connection.
func (q *Queue) Close() error {
	if err := q.client.Close(); err != nil {
		return err
	}
	return q.inspector.Close()
}

// RegisterJob registers a job type with the queue.
func (q *Queue) RegisterJob(job queue.Job) {
	q.registry.Register(job)
}

// GetRegistry returns the job registry.
func (q *Queue) GetRegistry() *queue.JobRegistry {
	return q.registry
}

// GetSerializer returns the serializer.
func (q *Queue) GetSerializer() *queue.Serializer {
	return q.serializer
}

// GetInspector returns the Asynq inspector for advanced operations.
func (q *Queue) GetInspector() *asynq.Inspector {
	return q.inspector
}

// buildTaskOptions converts queue options to Asynq options.
func (q *Queue) buildTaskOptions(job queue.Job, opts *queue.Options) []asynq.Option {
	asynqOpts := []asynq.Option{}

	// Queue name
	if opts.Queue != "" {
		asynqOpts = append(asynqOpts, asynq.Queue(opts.Queue))
	} else if jobQueue := job.Queue(); jobQueue != "" {
		asynqOpts = append(asynqOpts, asynq.Queue(jobQueue))
	}

	// Max retries
	maxRetries := opts.MaxRetries
	if maxRetries == 0 {
		maxRetries = job.Tries()
	}
	if maxRetries > 0 {
		asynqOpts = append(asynqOpts, asynq.MaxRetry(maxRetries))
	}

	// Timeout
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = job.Timeout()
	}
	if timeout > 0 {
		asynqOpts = append(asynqOpts, asynq.Timeout(timeout))
	}

	// Deadline
	if !opts.Deadline.IsZero() {
		asynqOpts = append(asynqOpts, asynq.Deadline(opts.Deadline))
	}

	// Delay (process at)
	if opts.Delay > 0 {
		asynqOpts = append(asynqOpts, asynq.ProcessIn(opts.Delay))
	}

	// Task ID
	if opts.TaskID != "" {
		asynqOpts = append(asynqOpts, asynq.TaskID(opts.TaskID))
	}

	// Retention
	if opts.Retention > 0 {
		asynqOpts = append(asynqOpts, asynq.Retention(opts.Retention))
	}

	// Group
	if opts.Group != "" {
		asynqOpts = append(asynqOpts, asynq.Group(opts.Group))
	}

	// Unique job handling
	if uniqueJob, ok := job.(queue.UniqueJob); ok {
		uniqueID := uniqueJob.UniqueID()
		uniqueTTL := uniqueJob.UniqueTTL()
		if uniqueTTL == 0 {
			uniqueTTL = 24 * time.Hour // Default 24 hours
		}
		asynqOpts = append(asynqOpts, asynq.Unique(uniqueTTL))
		if uniqueID != "" {
			asynqOpts = append(asynqOpts, asynq.TaskID(uniqueID))
		}
	}

	return asynqOpts
}

// convertTaskInfo converts Asynq task info to queue.JobInfo.
func (q *Queue) convertTaskInfo(info *asynq.TaskInfo) *queue.JobInfo {
	jobInfo := &queue.JobInfo{
		ID:            info.ID,
		Type:          info.Type,
		Queue:         info.Queue,
		MaxRetries:    info.MaxRetry,
		Retried:       info.Retried,
		NextProcessAt: info.NextProcessAt,
		LastError:     info.LastErr,
	}

	// Map Asynq state to queue state
	switch info.State {
	case asynq.TaskStatePending:
		jobInfo.State = queue.JobStatePending
	case asynq.TaskStateActive:
		jobInfo.State = queue.JobStateActive
	case asynq.TaskStateScheduled:
		jobInfo.State = queue.JobStateScheduled
	case asynq.TaskStateRetry:
		jobInfo.State = queue.JobStateRetry
	case asynq.TaskStateArchived:
		jobInfo.State = queue.JobStateFailed
	case asynq.TaskStateCompleted:
		jobInfo.State = queue.JobStateCompleted
		if !info.CompletedAt.IsZero() {
			completedAt := info.CompletedAt
			jobInfo.CompletedAt = &completedAt
		}
	}

	return jobInfo
}

// parseRedisConfig parses the connection config map into asynq.RedisClientOpt.
func parseRedisConfig(config map[string]any) asynq.RedisConnOpt {
	// Default values
	addr := "localhost:6379"
	password := ""
	db := 0

	// Extract configuration
	if val, ok := config["addr"].(string); ok {
		addr = val
	}
	if val, ok := config["password"].(string); ok {
		password = val
	}
	if val, ok := config["db"].(int); ok {
		db = val
	}

	// Check if using Redis cluster
	if addrs, ok := config["addrs"].([]string); ok && len(addrs) > 0 {
		return asynq.RedisClusterClientOpt{
			Addrs:    addrs,
			Password: password,
		}
	}

	// Check if using existing Redis client
	if client, ok := config["client"].(*redis.Client); ok {
		return asynq.RedisClientOpt{Addr: client.Options().Addr}
	}

	return asynq.RedisClientOpt{
		Addr:     addr,
		Password: password,
		DB:       db,
	}
}
