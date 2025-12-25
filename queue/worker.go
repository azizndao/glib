package queue

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// WorkerConfig contains configuration for a queue worker.
type WorkerConfig struct {
	// Connection is the queue connection name to use
	Connection string

	// Concurrency is the maximum number of concurrent workers
	Concurrency int

	// Queues is a map of queue names to their priority
	// Higher priority queues are processed first
	// Example: map[string]int{"high": 6, "default": 3, "low": 1}
	Queues map[string]int

	// StrictPriority controls whether higher priority queues are strictly prioritized
	// If true, jobs from lower priority queues will only be processed when
	// higher priority queues are empty
	StrictPriority bool

	// ShutdownTimeout is the timeout for graceful shutdown
	ShutdownTimeout time.Duration

	// HealthCheckInterval is the interval for health checks
	HealthCheckInterval time.Duration

	// Logger is the logger to use
	Logger *slog.Logger
}

// DefaultWorkerConfig returns a worker configuration with default values.
func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		Connection:          "default",
		Concurrency:         10,
		Queues:              map[string]int{"default": 1},
		StrictPriority:      false,
		ShutdownTimeout:     20 * time.Second,
		HealthCheckInterval: 15 * time.Second,
		Logger:              slog.Default(),
	}
}

// QueueWorker processes jobs from queues.
type QueueWorker struct {
	manager *Manager
	config  WorkerConfig
	worker  Worker
	jobs    []Job
}

// NewWorker creates a new queue worker.
func NewWorker(manager *Manager, config WorkerConfig) *QueueWorker {
	if manager == nil {
		manager = GetDefaultManager()
	}

	// Apply defaults
	if config.Concurrency == 0 {
		config.Concurrency = 10
	}
	if config.Queues == nil || len(config.Queues) == 0 {
		config.Queues = map[string]int{"default": 1}
	}
	if config.ShutdownTimeout == 0 {
		config.ShutdownTimeout = 20 * time.Second
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	if config.Connection == "" {
		config.Connection = "default"
	}

	return &QueueWorker{
		manager: manager,
		config:  config,
		jobs:    []Job{},
	}
}

// RegisterJob registers a job type that this worker can process.
func (w *QueueWorker) RegisterJob(job Job) {
	w.jobs = append(w.jobs, job)
	w.manager.RegisterJob(job)
}

// RegisterJobs registers multiple job types.
func (w *QueueWorker) RegisterJobs(jobs ...Job) {
	for _, job := range jobs {
		w.RegisterJob(job)
	}
}

// Start starts the worker to process jobs.
func (w *QueueWorker) Start(ctx context.Context) error {
	// Get the queue connection
	conn, err := w.manager.Connection(w.config.Connection)
	if err != nil {
		return fmt.Errorf("failed to get queue connection: %w", err)
	}

	// The actual worker implementation will vary by driver
	// For now, we check if it implements the Worker interface
	if worker, ok := conn.(Worker); ok {
		// Register all jobs
		for _, job := range w.jobs {
			jobType := w.manager.GetRegistry().TypeName(job)
			handler := func(ctx context.Context, j Job) error {
				return j.Handle(ctx)
			}
			worker.RegisterHandler(jobType, handler)
		}

		// Store the worker
		w.worker = worker

		// Start processing
		return worker.Start(ctx)
	}

	return fmt.Errorf("queue connection does not support workers")
}

// Stop stops the worker gracefully.
func (w *QueueWorker) Stop() error {
	if w.worker != nil {
		return w.worker.Stop()
	}
	return nil
}

// Work is a convenience method that starts the worker and blocks until
// the context is cancelled or an error occurs.
func (w *QueueWorker) Work(ctx context.Context) error {
	// Start the worker in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := w.Start(ctx); err != nil {
			errChan <- err
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		w.config.Logger.Info("Received shutdown signal")
		return w.Stop()
	case err := <-errChan:
		return err
	}
}
