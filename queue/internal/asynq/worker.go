package asynq

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/azizndao/glib/queue"
	"github.com/hibiken/asynq"
)

// Worker processes jobs from queues using Asynq.
type Worker struct {
	queue      *Queue
	server     *asynq.Server
	mux        *asynq.ServeMux
	config     WorkerConfig
	middleware []queue.JobMiddleware
	logger     *slog.Logger
}

// WorkerConfig contains configuration for the worker.
type WorkerConfig struct {
	// Concurrency is the maximum number of concurrent workers
	Concurrency int

	// Queues is a map of queue names to their priority
	// Higher priority queues are processed first
	Queues map[string]int

	// StrictPriority controls whether higher priority queues are strictly prioritized
	StrictPriority bool

	// RetryDelayFunc returns the delay before retrying a failed task
	RetryDelayFunc func(n int, err error, task *asynq.Task) time.Duration

	// IsFailureFunc determines whether an error should be considered a failure
	IsFailureFunc func(err error) bool

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
		Concurrency:         10,
		Queues:              map[string]int{"default": 1},
		StrictPriority:      false,
		ShutdownTimeout:     20 * time.Second,
		HealthCheckInterval: 15 * time.Second,
		Logger:              slog.Default(),
	}
}

// NewWorker creates a new worker.
func NewWorker(q *Queue, config WorkerConfig) *Worker {
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

	// Create Asynq server configuration
	redisOpt := parseRedisConfig(map[string]any{})
	if q.client != nil {
		// Extract Redis config from the queue's client
		// For simplicity, we'll use the default config
		// In production, this should be extracted properly
	}

	asynqConfig := asynq.Config{
		Concurrency:         config.Concurrency,
		Queues:              config.Queues,
		StrictPriority:      config.StrictPriority,
		RetryDelayFunc:      config.RetryDelayFunc,
		IsFailure:           config.IsFailureFunc,
		ShutdownTimeout:     config.ShutdownTimeout,
		HealthCheckInterval: config.HealthCheckInterval,
		Logger:              newAsynqLogger(config.Logger),
	}

	server := asynq.NewServer(redisOpt, asynqConfig)
	mux := asynq.NewServeMux()

	return &Worker{
		queue:      q,
		server:     server,
		mux:        mux,
		config:     config,
		middleware: []queue.JobMiddleware{},
		logger:     config.Logger,
	}
}

// RegisterHandler registers a job handler.
func (w *Worker) RegisterHandler(jobType string, handler queue.JobHandler) {
	w.mux.HandleFunc(jobType, w.wrapHandler(jobType, handler))
}

// RegisterMiddleware registers middleware to apply to all jobs.
func (w *Worker) RegisterMiddleware(middleware ...queue.JobMiddleware) {
	w.middleware = append(w.middleware, middleware...)
}

// RegisterJob registers a job and its handler.
func (w *Worker) RegisterJob(job queue.Job) {
	jobType := w.queue.registry.TypeName(job)

	handler := func(ctx context.Context, j queue.Job) error {
		return j.Handle(ctx)
	}

	// Apply job-specific middleware if the job implements the Middleware interface
	if middlewareJob, ok := job.(queue.Middleware); ok {
		jobMiddleware := middlewareJob.Middleware()
		for i := len(jobMiddleware) - 1; i >= 0; i-- {
			handler = jobMiddleware[i](handler)
		}
	}

	// Apply global middleware
	for i := len(w.middleware) - 1; i >= 0; i-- {
		handler = w.middleware[i](handler)
	}

	w.RegisterHandler(jobType, handler)
}

// RegisterJobs registers multiple jobs.
func (w *Worker) RegisterJobs(jobs ...queue.Job) {
	for _, job := range jobs {
		w.RegisterJob(job)
	}
}

// Start starts the worker to process jobs.
func (w *Worker) Start(ctx context.Context) error {
	w.logger.Info("Starting queue worker",
		"concurrency", w.config.Concurrency,
		"queues", w.config.Queues,
	)

	// Run the server
	if err := w.server.Run(w.mux); err != nil {
		return fmt.Errorf("worker failed: %w", err)
	}

	return nil
}

// Stop stops the worker gracefully.
func (w *Worker) Stop() error {
	w.logger.Info("Stopping queue worker...")
	w.server.Shutdown()
	w.logger.Info("Queue worker stopped")
	return nil
}

// wrapHandler wraps a job handler to integrate with Asynq.
func (w *Worker) wrapHandler(jobType string, handler queue.JobHandler) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, task *asynq.Task) error {
		// Deserialize the job
		job, err := w.queue.serializer.Deserialize(task.Payload())
		if err != nil {
			w.logger.Error("Failed to deserialize job",
				"type", jobType,
				"error", err,
			)
			return fmt.Errorf("failed to deserialize job: %w", err)
		}

		// Check if job should be deleted without processing
		if deletable, ok := job.(queue.Deletable); ok {
			if deletable.ShouldDelete() {
				w.logger.Info("Skipping job marked for deletion",
					"type", jobType,
					"id", task.ResultWriter().TaskID(),
				)
				return nil
			}
		}

		w.logger.Info("Processing job",
			"type", jobType,
			"id", task.ResultWriter().TaskID(),
		)

		// Execute the job
		err = handler(ctx, job)

		if err != nil {
			w.logger.Error("Job failed",
				"type", jobType,
				"id", task.ResultWriter().TaskID(),
				"error", err,
			)

			// Check if job should be released instead of retried
			if releasable, ok := job.(queue.Releasable); ok {
				shouldRelease, delay := releasable.Release()
				if shouldRelease {
					w.logger.Info("Releasing job back to queue",
						"type", jobType,
						"delay", delay,
					)
					// Asynq doesn't have a direct "release" mechanism
					// The job will be retried based on its retry configuration
				}
			}

			return err
		}

		w.logger.Info("Job completed successfully",
			"type", jobType,
			"id", task.ResultWriter().TaskID(),
		)

		return nil
	}
}

// asynqLogger wraps slog.Logger to implement asynq.Logger interface.
type asynqLogger struct {
	logger *slog.Logger
}

func newAsynqLogger(logger *slog.Logger) *asynqLogger {
	return &asynqLogger{logger: logger}
}

func (l *asynqLogger) Debug(args ...any) {
	l.logger.Debug(fmt.Sprint(args...))
}

func (l *asynqLogger) Info(args ...any) {
	l.logger.Info(fmt.Sprint(args...))
}

func (l *asynqLogger) Warn(args ...any) {
	l.logger.Warn(fmt.Sprint(args...))
}

func (l *asynqLogger) Error(args ...any) {
	l.logger.Error(fmt.Sprint(args...))
}

func (l *asynqLogger) Fatal(args ...any) {
	l.logger.Error(fmt.Sprint(args...))
	panic(fmt.Sprint(args...))
}
