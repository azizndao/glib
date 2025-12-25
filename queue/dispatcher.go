package queue

import (
	"context"
	"time"
)

// PendingDispatch represents a job that is ready to be dispatched.
// It provides a fluent interface for configuring job options.
type PendingDispatch struct {
	manager *Manager
	job     Job
	opts    *Options
}

// Dispatch creates a new pending dispatch for a job.
// This is the main entry point for dispatching jobs with a fluent API.
//
// Example:
//
//	Dispatch(&SendEmailJob{To: "user@example.com"}).
//	    OnQueue("emails").
//	    Delay(5 * time.Minute).
//	    Dispatch()
func Dispatch(job Job) *PendingDispatch {
	return &PendingDispatch{
		job:  job,
		opts: DefaultOptions(),
	}
}

// DispatchWith creates a pending dispatch using a specific manager.
func DispatchWith(manager *Manager, job Job) *PendingDispatch {
	return &PendingDispatch{
		manager: manager,
		job:     job,
		opts:    DefaultOptions(),
	}
}

// OnQueue specifies which queue the job should be pushed to.
func (pd *PendingDispatch) OnQueue(queue string) *PendingDispatch {
	pd.opts.Queue = queue
	return pd
}

// OnConnection specifies which connection the job should use.
func (pd *PendingDispatch) OnConnection(connection string) *PendingDispatch {
	pd.opts.Connection = connection
	return pd
}

// Delay specifies how long to wait before making the job available for processing.
func (pd *PendingDispatch) Delay(delay time.Duration) *PendingDispatch {
	pd.opts.Delay = delay
	return pd
}

// DelayUntil specifies when the job should become available for processing.
func (pd *PendingDispatch) DelayUntil(t time.Time) *PendingDispatch {
	pd.opts.Delay = time.Until(t)
	if pd.opts.Delay < 0 {
		pd.opts.Delay = 0
	}
	return pd
}

// MaxRetries specifies the maximum number of times the job should be retried.
func (pd *PendingDispatch) MaxRetries(retries int) *PendingDispatch {
	pd.opts.MaxRetries = retries
	return pd
}

// Timeout specifies the maximum time the job can run.
func (pd *PendingDispatch) Timeout(timeout time.Duration) *PendingDispatch {
	pd.opts.Timeout = timeout
	return pd
}

// Deadline specifies the absolute time by which the job must complete.
func (pd *PendingDispatch) Deadline(deadline time.Time) *PendingDispatch {
	pd.opts.Deadline = deadline
	return pd
}

// Retention specifies how long to keep the job after completion.
func (pd *PendingDispatch) Retention(retention time.Duration) *PendingDispatch {
	pd.opts.Retention = retention
	return pd
}

// UniqueFor specifies how long the job should remain unique.
// Only applicable if the job implements the UniqueJob interface.
func (pd *PendingDispatch) UniqueFor(duration time.Duration) *PendingDispatch {
	pd.opts.UniqueFor = duration
	return pd
}

// WithTaskID specifies a custom task ID for the job.
func (pd *PendingDispatch) WithTaskID(id string) *PendingDispatch {
	pd.opts.TaskID = id
	return pd
}

// InGroup specifies a group for the job.
func (pd *PendingDispatch) InGroup(group string) *PendingDispatch {
	pd.opts.Group = group
	return pd
}

// Dispatch dispatches the job to the queue and returns the job ID.
func (pd *PendingDispatch) Dispatch() (string, error) {
	return pd.DispatchContext(context.Background())
}

// DispatchContext dispatches the job with a context.
func (pd *PendingDispatch) DispatchContext(ctx context.Context) (string, error) {
	manager := pd.manager
	if manager == nil {
		manager = defaultManager
	}

	// Get the appropriate connection
	var queue Queue
	var err error

	if pd.opts.Connection != "" && pd.opts.Connection != "default" {
		queue, err = manager.Connection(pd.opts.Connection)
	} else {
		queue, err = manager.Default()
	}

	if err != nil {
		return "", err
	}

	// Dispatch the job
	return queue.Push(ctx, pd.job, pd.opts)
}

// DispatchIf dispatches the job only if the condition is true.
func (pd *PendingDispatch) DispatchIf(condition bool) (string, error) {
	if condition {
		return pd.Dispatch()
	}
	return "", nil
}

// DispatchUnless dispatches the job unless the condition is true.
func (pd *PendingDispatch) DispatchUnless(condition bool) (string, error) {
	return pd.DispatchIf(!condition)
}

// DispatchSync executes the job synchronously (for testing or immediate execution).
func (pd *PendingDispatch) DispatchSync() error {
	return pd.job.Handle(context.Background())
}

// Chain represents a chain of jobs that should execute sequentially.
type Chain struct {
	manager *Manager
	jobs    []Job
	opts    *Options
}

// NewChain creates a new job chain.
func NewChain(jobs ...Job) *Chain {
	return &Chain{
		jobs: jobs,
		opts: DefaultOptions(),
	}
}

// ChainWith creates a new job chain with a specific manager.
func ChainWith(manager *Manager, jobs ...Job) *Chain {
	return &Chain{
		manager: manager,
		jobs:    jobs,
		opts:    DefaultOptions(),
	}
}

// OnQueue specifies which queue the chain should use.
func (c *Chain) OnQueue(queue string) *Chain {
	c.opts.Queue = queue
	return c
}

// OnConnection specifies which connection the chain should use.
func (c *Chain) OnConnection(connection string) *Chain {
	c.opts.Connection = connection
	return c
}

// Dispatch dispatches the job chain.
// The jobs will execute sequentially, with each job only running if the previous one succeeded.
func (c *Chain) Dispatch() ([]string, error) {
	return c.DispatchContext(context.Background())
}

// DispatchContext dispatches the job chain with a context.
func (c *Chain) DispatchContext(ctx context.Context) ([]string, error) {
	manager := c.manager
	if manager == nil {
		manager = defaultManager
	}

	// Get the queue
	var queue Queue
	var err error

	if c.opts.Connection != "" && c.opts.Connection != "default" {
		queue, err = manager.Connection(c.opts.Connection)
	} else {
		queue, err = manager.Default()
	}

	if err != nil {
		return nil, err
	}

	// For now, dispatch all jobs independently
	// TODO: Implement proper job chaining with dependencies
	ids := make([]string, 0, len(c.jobs))
	for _, job := range c.jobs {
		id, err := queue.Push(ctx, job, c.opts)
		if err != nil {
			return ids, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// Batch represents a batch of jobs that can execute in parallel.
type Batch struct {
	manager *Manager
	jobs    []Job
	opts    *Options
}

// NewBatch creates a new job batch.
func NewBatch(jobs ...Job) *Batch {
	return &Batch{
		jobs: jobs,
		opts: DefaultOptions(),
	}
}

// BatchWith creates a new job batch with a specific manager.
func BatchWith(manager *Manager, jobs ...Job) *Batch {
	return &Batch{
		manager: manager,
		jobs:    jobs,
		opts:    DefaultOptions(),
	}
}

// OnQueue specifies which queue the batch should use.
func (b *Batch) OnQueue(queue string) *Batch {
	b.opts.Queue = queue
	return b
}

// OnConnection specifies which connection the batch should use.
func (b *Batch) OnConnection(connection string) *Batch {
	b.opts.Connection = connection
	return b
}

// Dispatch dispatches all jobs in the batch.
func (b *Batch) Dispatch() ([]string, error) {
	return b.DispatchContext(context.Background())
}

// DispatchContext dispatches all jobs in the batch with a context.
func (b *Batch) DispatchContext(ctx context.Context) ([]string, error) {
	manager := b.manager
	if manager == nil {
		manager = defaultManager
	}

	// Get the queue
	var queue Queue
	var err error

	if b.opts.Connection != "" && b.opts.Connection != "default" {
		queue, err = manager.Connection(b.opts.Connection)
	} else {
		queue, err = manager.Default()
	}

	if err != nil {
		return nil, err
	}

	// Dispatch all jobs
	ids := make([]string, 0, len(b.jobs))
	for _, job := range b.jobs {
		id, err := queue.Push(ctx, job, b.opts)
		if err != nil {
			return ids, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// Global default manager for convenience functions
var defaultManager *Manager

// SetDefaultManager sets the default manager for convenience functions.
func SetDefaultManager(manager *Manager) {
	defaultManager = manager
}

// GetDefaultManager returns the default manager.
func GetDefaultManager() *Manager {
	if defaultManager == nil {
		defaultManager = NewManager()
	}
	return defaultManager
}
