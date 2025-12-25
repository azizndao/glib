package queue

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
)

// JobRegistry manages the registration and instantiation of job types.
type JobRegistry struct {
	types map[string]reflect.Type
	mu    sync.RWMutex
}

// NewJobRegistry creates a new job registry.
func NewJobRegistry() *JobRegistry {
	return &JobRegistry{
		types: make(map[string]reflect.Type),
	}
}

// Register registers a job type in the registry.
// The job parameter should be a pointer to a job instance (e.g., &MyJob{}).
func (r *JobRegistry) Register(job Job) {
	r.mu.Lock()
	defer r.mu.Unlock()

	t := reflect.TypeOf(job)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	typeName := t.PkgPath() + "." + t.Name()
	r.types[typeName] = t
}

// Get returns the reflect.Type for a given type name.
func (r *JobRegistry) Get(typeName string) (reflect.Type, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, exists := r.types[typeName]
	return t, exists
}

// TypeName returns the full type name for a job.
func (r *JobRegistry) TypeName(job Job) string {
	t := reflect.TypeOf(job)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.PkgPath() + "." + t.Name()
}

// NewInstance creates a new instance of a job type.
func (r *JobRegistry) NewInstance(typeName string) (Job, error) {
	t, exists := r.Get(typeName)
	if !exists {
		return nil, fmt.Errorf("%w: %s not registered", ErrJobNotFound, typeName)
	}

	// Create a pointer to a new instance
	ptr := reflect.New(t)
	job, ok := ptr.Interface().(Job)
	if !ok {
		return nil, fmt.Errorf("%w: %s does not implement Job interface", ErrInvalidJob, typeName)
	}

	return job, nil
}

// Payload represents the serialized job data.
type Payload struct {
	// Type is the full type name of the job
	Type string `json:"type"`

	// Data is the serialized job data
	Data []byte `json:"data"`

	// Format is the serialization format ("gob" or "json")
	Format string `json:"format"`
}

// Serializer handles job serialization and deserialization.
type Serializer struct {
	registry *JobRegistry
	format   string // "gob" or "json"
}

// NewSerializer creates a new job serializer.
// format can be "gob" (default, fast and type-safe) or "json" (slower but more portable).
func NewSerializer(registry *JobRegistry, format string) *Serializer {
	if format == "" {
		format = "gob"
	}
	return &Serializer{
		registry: registry,
		format:   format,
	}
}

// Serialize serializes a job into bytes.
func (s *Serializer) Serialize(job Job) ([]byte, error) {
	typeName := s.registry.TypeName(job)

	var data []byte
	var err error

	switch s.format {
	case "gob":
		data, err = s.serializeGob(job)
	case "json":
		data, err = s.serializeJSON(job)
	default:
		return nil, fmt.Errorf("unsupported serialization format: %s", s.format)
	}

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrJobSerializationFailed, err)
	}

	payload := Payload{
		Type:   typeName,
		Data:   data,
		Format: s.format,
	}

	// Always use JSON for the outer payload structure
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to marshal payload: %v", ErrJobSerializationFailed, err)
	}

	return payloadBytes, nil
}

// Deserialize deserializes bytes into a job.
func (s *Serializer) Deserialize(data []byte) (Job, error) {
	// Unmarshal the outer payload structure
	var payload Payload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal payload: %v", ErrJobDeserializationFailed, err)
	}

	// Create a new instance of the job type
	job, err := s.registry.NewInstance(payload.Type)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrJobDeserializationFailed, err)
	}

	// Deserialize the job data into the instance
	switch payload.Format {
	case "gob":
		err = s.deserializeGob(payload.Data, job)
	case "json":
		err = s.deserializeJSON(payload.Data, job)
	default:
		return nil, fmt.Errorf("unsupported deserialization format: %s", payload.Format)
	}

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrJobDeserializationFailed, err)
	}

	return job, nil
}

// serializeGob serializes a job using gob encoding.
func (s *Serializer) serializeGob(job Job) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(job); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// deserializeGob deserializes a job using gob decoding.
func (s *Serializer) deserializeGob(data []byte, job Job) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(job)
}

// serializeJSON serializes a job using JSON encoding.
func (s *Serializer) serializeJSON(job Job) ([]byte, error) {
	return json.Marshal(job)
}

// deserializeJSON deserializes a job using JSON decoding.
func (s *Serializer) deserializeJSON(data []byte, job Job) error {
	return json.Unmarshal(data, job)
}

// Global registry for convenience
var globalRegistry = NewJobRegistry()

// Register registers a job type in the global registry.
// This should be called in init() functions.
//
// Example:
//
//	func init() {
//	    queue.Register(&MyJob{})
//	}
func Register(job Job) {
	globalRegistry.Register(job)
}

// TypeName returns the type name for a job using the global registry.
func TypeName(job Job) string {
	return globalRegistry.TypeName(job)
}

// NewGlobalSerializer creates a new serializer using the global registry.
func NewGlobalSerializer(format string) *Serializer {
	return NewSerializer(globalRegistry, format)
}
