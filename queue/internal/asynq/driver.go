package asynq

import (
	"github.com/azizndao/glib/queue"
)

// NewQueue creates a new Asynq queue driver.
func NewQueue(config queue.Config) (queue.Queue, error) {
	return New(config)
}

// NewQueueWorker creates a worker for an Asynq queue.
func NewQueueWorker(q queue.Queue, config queue.WorkerConfig) (queue.Worker, error) {
	asynqQueue, ok := q.(*Queue)
	if !ok {
		return nil, queue.ErrInvalidJob
	}

	workerConfig := WorkerConfig{
		Concurrency:         config.Concurrency,
		Queues:              config.Queues,
		StrictPriority:      config.StrictPriority,
		ShutdownTimeout:     config.ShutdownTimeout,
		HealthCheckInterval: config.HealthCheckInterval,
		Logger:              config.Logger,
	}

	return NewWorker(asynqQueue, workerConfig), nil
}
