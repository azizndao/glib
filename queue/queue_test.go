package queue

import (
	"context"
	"testing"
	"time"
)

// TestJob is a test job
type TestJob struct {
	BaseJob
	Data string
}

func (j *TestJob) Handle(ctx context.Context) error {
	return nil
}

func TestJobInterface(t *testing.T) {
	job := &TestJob{Data: "test"}

	// Test BaseJob defaults
	if job.Queue() != "" {
		t.Errorf("Expected empty queue, got %s", job.Queue())
	}

	if job.Tries() != 1 {
		t.Errorf("Expected 1 try, got %d", job.Tries())
	}

	if job.Timeout() != 0 {
		t.Errorf("Expected 0 timeout, got %v", job.Timeout())
	}

	if job.RetryDelay(1) != 0 {
		t.Errorf("Expected 0 retry delay, got %v", job.RetryDelay(1))
	}

	// Test Handle
	err := job.Handle(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test Failed (should not panic)
	job.Failed(context.Background(), nil)
}

func TestJobRegistry(t *testing.T) {
	registry := NewJobRegistry()

	// Register a job
	job := &TestJob{}
	registry.Register(job)

	// Get type name
	typeName := registry.TypeName(job)
	if typeName == "" {
		t.Error("Expected non-empty type name")
	}

	// Get registered type
	typ, exists := registry.Get(typeName)
	if !exists {
		t.Errorf("Expected type %s to be registered", typeName)
	}

	if typ.Name() != "TestJob" {
		t.Errorf("Expected type name TestJob, got %s", typ.Name())
	}

	// Create new instance
	newJob, err := registry.NewInstance(typeName)
	if err != nil {
		t.Fatalf("Failed to create new instance: %v", err)
	}

	if _, ok := newJob.(*TestJob); !ok {
		t.Error("Expected *TestJob type")
	}
}

func TestSerializer(t *testing.T) {
	registry := NewJobRegistry()
	registry.Register(&TestJob{})

	serializer := NewSerializer(registry, "gob")

	job := &TestJob{Data: "test data"}

	// Serialize
	data, err := serializer.Serialize(job)
	if err != nil {
		t.Fatalf("Failed to serialize job: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty serialized data")
	}

	// Deserialize
	deserializedJob, err := serializer.Deserialize(data)
	if err != nil {
		t.Fatalf("Failed to deserialize job: %v", err)
	}

	testJob, ok := deserializedJob.(*TestJob)
	if !ok {
		t.Fatal("Expected *TestJob type")
	}

	if testJob.Data != "test data" {
		t.Errorf("Expected 'test data', got '%s'", testJob.Data)
	}
}

func TestSerializerJSON(t *testing.T) {
	registry := NewJobRegistry()
	registry.Register(&TestJob{})

	serializer := NewSerializer(registry, "json")

	job := &TestJob{Data: "test data"}

	// Serialize
	data, err := serializer.Serialize(job)
	if err != nil {
		t.Fatalf("Failed to serialize job: %v", err)
	}

	// Deserialize
	deserializedJob, err := serializer.Deserialize(data)
	if err != nil {
		t.Fatalf("Failed to deserialize job: %v", err)
	}

	testJob, ok := deserializedJob.(*TestJob)
	if !ok {
		t.Fatal("Expected *TestJob type")
	}

	if testJob.Data != "test data" {
		t.Errorf("Expected 'test data', got '%s'", testJob.Data)
	}
}

func TestManager(t *testing.T) {
	manager := NewManager()

	// Test default connection name
	manager.SetDefaultConnection("test")

	// Register a mock driver
	driverCalled := false
	mockDriver := func(config Config) (Queue, error) {
		driverCalled = true
		return nil, nil
	}

	manager.RegisterDriver("mock", mockDriver)

	// Register config
	manager.RegisterConfig("test", Config{
		Driver: "mock",
		Connection: map[string]any{
			"addr": "localhost:6379",
		},
	})

	// Test connection (will fail because mock returns nil, but driver should be called)
	_, _ = manager.Connection("test")

	if !driverCalled {
		t.Error("Expected driver to be called")
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.Queue != "default" {
		t.Errorf("Expected default queue, got %s", opts.Queue)
	}

	if opts.Connection != "default" {
		t.Errorf("Expected default connection, got %s", opts.Connection)
	}

	if opts.MaxRetries != 1 {
		t.Errorf("Expected 1 max retry, got %d", opts.MaxRetries)
	}

	if opts.TaskID == "" {
		t.Error("Expected non-empty task ID")
	}
}

func TestPendingDispatch(t *testing.T) {
	job := &TestJob{Data: "test"}
	pd := Dispatch(job)

	// Test fluent API
	pd.OnQueue("test-queue").
		Delay(5 * time.Second).
		MaxRetries(3).
		Timeout(30 * time.Second)

	if pd.opts.Queue != "test-queue" {
		t.Errorf("Expected queue 'test-queue', got '%s'", pd.opts.Queue)
	}

	if pd.opts.Delay != 5*time.Second {
		t.Errorf("Expected 5s delay, got %v", pd.opts.Delay)
	}

	if pd.opts.MaxRetries != 3 {
		t.Errorf("Expected 3 max retries, got %d", pd.opts.MaxRetries)
	}

	if pd.opts.Timeout != 30*time.Second {
		t.Errorf("Expected 30s timeout, got %v", pd.opts.Timeout)
	}
}

func TestEventDispatcher(t *testing.T) {
	dispatcher := NewEventDispatcher()

	// Create and dispatch using an actual queue event
	listenerCalled := false
	dispatcher.Listen("job.dispatched", func(ctx context.Context, event Event) error {
		listenerCalled = true
		e, ok := event.(*JobDispatchedEvent)
		if !ok {
			t.Error("Expected JobDispatchedEvent")
		}
		if e.JobID != "test-123" {
			t.Errorf("Expected job ID test-123, got %s", e.JobID)
		}
		return nil
	})

	event := &JobDispatchedEvent{
		JobID:      "test-123",
		JobType:    "TestJob",
		Queue:      "default",
		Connection: "default",
	}

	err := dispatcher.Dispatch(context.Background(), event)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !listenerCalled {
		t.Error("Expected listener to be called")
	}
}

func TestChain(t *testing.T) {
	jobs := []Job{
		&TestJob{Data: "job1"},
		&TestJob{Data: "job2"},
		&TestJob{Data: "job3"},
	}

	chain := NewChain(jobs...)

	if len(chain.jobs) != 3 {
		t.Errorf("Expected 3 jobs, got %d", len(chain.jobs))
	}

	chain.OnQueue("test").OnConnection("redis")

	if chain.opts.Queue != "test" {
		t.Errorf("Expected queue 'test', got '%s'", chain.opts.Queue)
	}

	if chain.opts.Connection != "redis" {
		t.Errorf("Expected connection 'redis', got '%s'", chain.opts.Connection)
	}
}

func TestBatch(t *testing.T) {
	jobs := []Job{
		&TestJob{Data: "job1"},
		&TestJob{Data: "job2"},
	}

	batch := NewBatch(jobs...)

	if len(batch.jobs) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(batch.jobs))
	}

	batch.OnQueue("test")

	if batch.opts.Queue != "test" {
		t.Errorf("Expected queue 'test', got '%s'", batch.opts.Queue)
	}
}
