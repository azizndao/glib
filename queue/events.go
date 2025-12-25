package queue

import (
	"context"
	"sync"
)

// Event represents a queue event.
type Event interface {
	// EventName returns the name of the event
	EventName() string
}

// EventListener is a function that handles an event.
type EventListener func(ctx context.Context, event Event) error

// EventDispatcher manages event listeners and dispatches events.
type EventDispatcher struct {
	listeners map[string][]EventListener
	mu        sync.RWMutex
}

// NewEventDispatcher creates a new event dispatcher.
func NewEventDispatcher() *EventDispatcher {
	return &EventDispatcher{
		listeners: make(map[string][]EventListener),
	}
}

// Listen registers an event listener.
func (ed *EventDispatcher) Listen(eventName string, listener EventListener) {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	ed.listeners[eventName] = append(ed.listeners[eventName], listener)
}

// Dispatch dispatches an event to all registered listeners.
func (ed *EventDispatcher) Dispatch(ctx context.Context, event Event) error {
	ed.mu.RLock()
	listeners := ed.listeners[event.EventName()]
	ed.mu.RUnlock()

	for _, listener := range listeners {
		if err := listener(ctx, event); err != nil {
			return err
		}
	}

	return nil
}

// Common queue events

// JobDispatchedEvent is fired when a job is dispatched to the queue.
type JobDispatchedEvent struct {
	JobID      string
	JobType    string
	Queue      string
	Connection string
}

func (e *JobDispatchedEvent) EventName() string {
	return "job.dispatched"
}

// JobProcessingEvent is fired when a job starts processing.
type JobProcessingEvent struct {
	JobID   string
	JobType string
	Queue   string
}

func (e *JobProcessingEvent) EventName() string {
	return "job.processing"
}

// JobProcessedEvent is fired when a job completes successfully.
type JobProcessedEvent struct {
	JobID   string
	JobType string
	Queue   string
}

func (e *JobProcessedEvent) EventName() string {
	return "job.processed"
}

// JobFailedEvent is fired when a job fails.
type JobFailedEvent struct {
	JobID   string
	JobType string
	Queue   string
	Error   error
	Attempt int
}

func (e *JobFailedEvent) EventName() string {
	return "job.failed"
}

// JobRetryingEvent is fired when a job is being retried.
type JobRetryingEvent struct {
	JobID   string
	JobType string
	Queue   string
	Attempt int
}

func (e *JobRetryingEvent) EventName() string {
	return "job.retrying"
}

// Global event dispatcher
var globalEventDispatcher = NewEventDispatcher()

// Listen registers a global event listener.
func Listen(eventName string, listener EventListener) {
	globalEventDispatcher.Listen(eventName, listener)
}

// DispatchEvent dispatches an event using the global dispatcher.
func DispatchEvent(ctx context.Context, event Event) error {
	return globalEventDispatcher.Dispatch(ctx, event)
}
