# Phase 7: Developer Experience Enhancements

**Timeline**: Weeks 19-20  
**Priority**: High - Critical for adoption and productivity  
**Dependencies**: Phase 1 (Foundation), Phase 2 (Database)

## Overview

Build developer-focused utilities and tools inspired by Laravel's Collections, Model Factories, and Testing helpers:
- Generic Collections API for data manipulation
- Model Factories for test data generation
- Database Seeders for populating test/development data
- HTTP testing utilities
- Database testing helpers
- Fake implementations for external services
- Assertion helpers

## Package Structure

```
collections/
├── collection.go      # Generic collection implementation
├── map.go            # Map operations
├── reduce.go         # Reduce operations
└── helpers.go        # Helper functions

testing/
├── http.go           # HTTP testing utilities
├── database.go       # Database testing helpers
├── assertions.go     # Custom assertions
└── mocks.go          # Mock implementations

factories/
├── factory.go        # Factory interface
├── builder.go        # Factory builder
└── faker.go          # Fake data generation

seeders/
├── seeder.go         # Seeder interface
└── runner.go         # Seeder runner

fakes/
├── cache.go          # Fake cache implementation
├── storage.go        # Fake storage implementation
├── queue.go          # Fake queue implementation
└── mailer.go         # Fake mailer implementation
```

## 1. Collections API

### Collection Interface

```go
package collections

import "iter"

// Collection represents a generic collection of items
type Collection[T any] struct {
	items []T
}

// New creates a new collection
func New[T any](items ...T) *Collection[T] {
	return &Collection[T]{items: items}
}

// From creates a collection from a slice
func From[T any](items []T) *Collection[T] {
	return &Collection[T]{items: items}
}

// All returns all items
func (c *Collection[T]) All() []T {
	return c.items
}

// Count returns the number of items
func (c *Collection[T]) Count() int {
	return len(c.items)
}

// IsEmpty checks if collection is empty
func (c *Collection[T]) IsEmpty() bool {
	return len(c.items) == 0
}

// IsNotEmpty checks if collection is not empty
func (c *Collection[T]) IsNotEmpty() bool {
	return len(c.items) > 0
}

// First returns the first item
func (c *Collection[T]) First() (T, bool) {
	if len(c.items) == 0 {
		var zero T
		return zero, false
	}
	return c.items[0], true
}

// Last returns the last item
func (c *Collection[T]) Last() (T, bool) {
	if len(c.items) == 0 {
		var zero T
		return zero, false
	}
	return c.items[len(c.items)-1], true
}

// Get returns item at index
func (c *Collection[T]) Get(index int) (T, bool) {
	if index < 0 || index >= len(c.items) {
		var zero T
		return zero, false
	}
	return c.items[index], true
}

// Push adds item to end
func (c *Collection[T]) Push(item T) *Collection[T] {
	c.items = append(c.items, item)
	return c
}

// Prepend adds item to beginning
func (c *Collection[T]) Prepend(item T) *Collection[T] {
	c.items = append([]T{item}, c.items...)
	return c
}

// Pop removes and returns last item
func (c *Collection[T]) Pop() (T, bool) {
	if len(c.items) == 0 {
		var zero T
		return zero, false
	}
	item := c.items[len(c.items)-1]
	c.items = c.items[:len(c.items)-1]
	return item, true
}

// Shift removes and returns first item
func (c *Collection[T]) Shift() (T, bool) {
	if len(c.items) == 0 {
		var zero T
		return zero, false
	}
	item := c.items[0]
	c.items = c.items[1:]
	return item, true
}

// Filter filters items by predicate
func (c *Collection[T]) Filter(fn func(T) bool) *Collection[T] {
	result := make([]T, 0, len(c.items))
	for _, item := range c.items {
		if fn(item) {
			result = append(result, item)
		}
	}
	return &Collection[T]{items: result}
}

// Map transforms items
func Map[T, U any](c *Collection[T], fn func(T) U) *Collection[U] {
	result := make([]U, len(c.items))
	for i, item := range c.items {
		result[i] = fn(item)
	}
	return &Collection[U]{items: result}
}

// Reduce reduces collection to single value
func Reduce[T, U any](c *Collection[T], fn func(U, T) U, initial U) U {
	result := initial
	for _, item := range c.items {
		result = fn(result, item)
	}
	return result
}

// Each iterates over items
func (c *Collection[T]) Each(fn func(T)) *Collection[T] {
	for _, item := range c.items {
		fn(item)
	}
	return c
}

// EachWithIndex iterates with index
func (c *Collection[T]) EachWithIndex(fn func(int, T)) *Collection[T] {
	for i, item := range c.items {
		fn(i, item)
	}
	return c
}

// Chunk splits collection into chunks
func (c *Collection[T]) Chunk(size int) [][]*Collection[T] {
	var chunks [][]*Collection[T]
	for i := 0; i < len(c.items); i += size {
		end := i + size
		if end > len(c.items) {
			end = len(c.items)
		}
		chunks = append(chunks, []*Collection[T]{From(c.items[i:end])})
	}
	return chunks
}

// Contains checks if item exists
func (c *Collection[T]) Contains(fn func(T) bool) bool {
	for _, item := range c.items {
		if fn(item) {
			return true
		}
	}
	return false
}

// Find finds first matching item
func (c *Collection[T]) Find(fn func(T) bool) (T, bool) {
	for _, item := range c.items {
		if fn(item) {
			return item, true
		}
	}
	var zero T
	return zero, false
}

// Where filters by field value
func Where[T any, V comparable](c *Collection[T], field func(T) V, value V) *Collection[T] {
	return c.Filter(func(item T) bool {
		return field(item) == value
	})
}

// Unique returns unique items
func (c *Collection[T]) Unique(key func(T) any) *Collection[T] {
	seen := make(map[any]bool)
	result := make([]T, 0)
	
	for _, item := range c.items {
		k := key(item)
		if !seen[k] {
			seen[k] = true
			result = append(result, item)
		}
	}
	
	return &Collection[T]{items: result}
}

// GroupBy groups items by key
func GroupBy[T any, K comparable](c *Collection[T], key func(T) K) map[K]*Collection[T] {
	groups := make(map[K]*Collection[T])
	
	for _, item := range c.items {
		k := key(item)
		if _, exists := groups[k]; !exists {
			groups[k] = New[T]()
		}
		groups[k].Push(item)
	}
	
	return groups
}

// SortBy sorts by key function
func SortBy[T any, K any](c *Collection[T], key func(T) K, less func(K, K) bool) *Collection[T] {
	items := make([]T, len(c.items))
	copy(items, c.items)
	
	sort.Slice(items, func(i, j int) bool {
		return less(key(items[i]), key(items[j]))
	})
	
	return &Collection[T]{items: items}
}

// Reverse reverses the collection
func (c *Collection[T]) Reverse() *Collection[T] {
	result := make([]T, len(c.items))
	for i, item := range c.items {
		result[len(c.items)-1-i] = item
	}
	return &Collection[T]{items: result}
}

// Take returns first n items
func (c *Collection[T]) Take(n int) *Collection[T] {
	if n > len(c.items) {
		n = len(c.items)
	}
	return &Collection[T]{items: c.items[:n]}
}

// Skip skips first n items
func (c *Collection[T]) Skip(n int) *Collection[T] {
	if n > len(c.items) {
		return &Collection[T]{items: []T{}}
	}
	return &Collection[T]{items: c.items[n:]}
}

// Slice returns a slice of items
func (c *Collection[T]) Slice(start, end int) *Collection[T] {
	if start < 0 {
		start = 0
	}
	if end > len(c.items) {
		end = len(c.items)
	}
	return &Collection[T]{items: c.items[start:end]}
}

// Pluck extracts field values
func Pluck[T any, V any](c *Collection[T], field func(T) V) *Collection[V] {
	result := make([]V, len(c.items))
	for i, item := range c.items {
		result[i] = field(item)
	}
	return &Collection[V]{items: result}
}

// Sum sums numeric values
func Sum[T any, V constraints.Integer | constraints.Float](c *Collection[T], field func(T) V) V {
	var sum V
	for _, item := range c.items {
		sum += field(item)
	}
	return sum
}

// Avg calculates average
func Avg[T any, V constraints.Integer | constraints.Float](c *Collection[T], field func(T) V) float64 {
	if len(c.items) == 0 {
		return 0
	}
	sum := Sum(c, field)
	return float64(sum) / float64(len(c.items))
}

// Min finds minimum value
func Min[T any, V constraints.Ordered](c *Collection[T], field func(T) V) V {
	if len(c.items) == 0 {
		var zero V
		return zero
	}
	
	min := field(c.items[0])
	for _, item := range c.items[1:] {
		val := field(item)
		if val < min {
			min = val
		}
	}
	return min
}

// Max finds maximum value
func Max[T any, V constraints.Ordered](c *Collection[T], field func(T) V) V {
	if len(c.items) == 0 {
		var zero V
		return zero
	}
	
	max := field(c.items[0])
	for _, item := range c.items[1:] {
		val := field(item)
		if val > max {
			max = val
		}
	}
	return max
}

// ToMap converts collection to map
func ToMap[T any, K comparable, V any](c *Collection[T], key func(T) K, value func(T) V) map[K]V {
	result := make(map[K]V, len(c.items))
	for _, item := range c.items {
		result[key(item)] = value(item)
	}
	return result
}

// Flatten flattens nested collections
func Flatten[T any](c *Collection[[]T]) *Collection[T] {
	result := make([]T, 0)
	for _, items := range c.items {
		result = append(result, items...)
	}
	return &Collection[T]{items: result}
}

// Join joins items into string
func Join[T any](c *Collection[T], separator string, toString func(T) string) string {
	if len(c.items) == 0 {
		return ""
	}
	
	parts := make([]string, len(c.items))
	for i, item := range c.items {
		parts[i] = toString(item)
	}
	
	return strings.Join(parts, separator)
}
```

## 2. Model Factories

### Factory Interface

```go
package factories

import (
	"context"
	"math/rand"
	
	"github.com/brianvoe/gofakeit/v6"
)

// Factory represents a model factory
type Factory[T any] interface {
	// Definition returns the default attribute values
	Definition() map[string]any
	
	// Make creates a model instance without saving
	Make(attributes ...map[string]any) T
	
	// Create creates and saves a model instance
	Create(ctx context.Context, attributes ...map[string]any) (T, error)
	
	// CreateMany creates multiple instances
	CreateMany(ctx context.Context, count int, attributes ...map[string]any) ([]T, error)
	
	// State applies a state modifier
	State(name string, modifier func(map[string]any)) Factory[T]
	
	// Count sets the number of instances to create
	Count(n int) Factory[T]
	
	// Sequence creates a sequence for attributes
	Sequence(field string, fn func(int) any) Factory[T]
}

// BaseFactory provides common factory functionality
type BaseFactory[T any] struct {
	definition map[string]any
	states     map[string]func(map[string]any)
	sequences  map[string]func(int) any
	count      int
	faker      *gofakeit.Faker
}

// NewFactory creates a new factory
func NewFactory[T any](definition map[string]any) *BaseFactory[T] {
	return &BaseFactory[T]{
		definition: definition,
		states:     make(map[string]func(map[string]any)),
		sequences:  make(map[string]func(int) any),
		count:      1,
		faker:      gofakeit.New(0),
	}
}

// State adds a state modifier
func (f *BaseFactory[T]) State(name string, modifier func(map[string]any)) Factory[T] {
	f.states[name] = modifier
	return f
}

// Count sets the count
func (f *BaseFactory[T]) Count(n int) Factory[T] {
	f.count = n
	return f
}

// Sequence adds a sequence
func (f *BaseFactory[T]) Sequence(field string, fn func(int) any) Factory[T] {
	f.sequences[field] = fn
	return f
}

// Make creates an instance without saving
func (f *BaseFactory[T]) Make(attributes ...map[string]any) T {
	attrs := f.mergeAttributes(0, attributes...)
	return f.build(attrs)
}

// mergeAttributes merges definition, states, sequences, and overrides
func (f *BaseFactory[T]) mergeAttributes(index int, overrides ...map[string]any) map[string]any {
	// Start with definition
	attrs := make(map[string]any)
	for k, v := range f.definition {
		attrs[k] = v
	}
	
	// Apply sequences
	for field, fn := range f.sequences {
		attrs[field] = fn(index)
	}
	
	// Apply states
	for _, modifier := range f.states {
		modifier(attrs)
	}
	
	// Apply overrides
	for _, override := range overrides {
		for k, v := range override {
			attrs[k] = v
		}
	}
	
	return attrs
}

// build constructs the model from attributes
func (f *BaseFactory[T]) build(attrs map[string]any) T {
	var model T
	// Use reflection to set fields from attrs
	// Implementation depends on your struct mapping strategy
	return model
}
```

### Example Factory Implementation

```go
package factories

import (
	"context"
	"time"
	
	"github.com/yourusername/glib/database"
)

// User model
type User struct {
	ID        uint
	Name      string
	Email     string
	Password  string
	IsAdmin   bool
	CreatedAt time.Time
}

// UserFactory creates User instances
type UserFactory struct {
	*BaseFactory[User]
	db *database.Manager
}

// NewUserFactory creates a new user factory
func NewUserFactory(db *database.Manager) *UserFactory {
	base := NewFactory[User](map[string]any{
		"name":     func() any { return gofakeit.Name() },
		"email":    func() any { return gofakeit.Email() },
		"password": "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", // "password"
		"is_admin": false,
	})
	
	factory := &UserFactory{
		BaseFactory: base,
		db:          db,
	}
	
	// Define states
	factory.State("admin", func(attrs map[string]any) {
		attrs["is_admin"] = true
	})
	
	factory.State("unverified", func(attrs map[string]any) {
		attrs["email_verified_at"] = nil
	})
	
	return factory
}

// Create creates and saves a user
func (f *UserFactory) Create(ctx context.Context, attributes ...map[string]any) (User, error) {
	user := f.Make(attributes...)
	
	conn, err := f.db.Connection("")
	if err != nil {
		return User{}, err
	}
	
	if err := conn.Create(&user).Error; err != nil {
		return User{}, err
	}
	
	return user, nil
}

// CreateMany creates multiple users
func (f *UserFactory) CreateMany(ctx context.Context, count int, attributes ...map[string]any) ([]User, error) {
	users := make([]User, count)
	
	for i := 0; i < count; i++ {
		attrs := f.mergeAttributes(i, attributes...)
		users[i] = f.build(attrs)
	}
	
	conn, err := f.db.Connection("")
	if err != nil {
		return nil, err
	}
	
	if err := conn.Create(&users).Error; err != nil {
		return nil, err
	}
	
	return users, nil
}
```

### Factory Usage Examples

```go
package main

import (
	"context"
	
	"github.com/yourusername/glib/factories"
)

func SeedDatabase(userFactory *factories.UserFactory) error {
	ctx := context.Background()
	
	// Create a single user
	user, err := userFactory.Create(ctx)
	
	// Create with custom attributes
	admin, err := userFactory.State("admin").Create(ctx, map[string]any{
		"name":  "Admin User",
		"email": "admin@example.com",
	})
	
	// Create multiple users
	users, err := userFactory.Count(10).Create(ctx)
	
	// Create users with sequence
	userFactory.Sequence("email", func(i int) any {
		return fmt.Sprintf("user%d@example.com", i)
	}).CreateMany(ctx, 5)
	
	// Make without saving
	user := userFactory.Make(map[string]any{
		"name": "Test User",
	})
	
	return nil
}
```

## 3. Database Seeders

```go
package seeders

import (
	"context"
	
	"github.com/yourusername/glib/database"
)

// Seeder represents a database seeder
type Seeder interface {
	Run(ctx context.Context) error
}

// Runner runs database seeders
type Runner struct {
	db      *database.Manager
	seeders []Seeder
}

// NewRunner creates a new seeder runner
func NewRunner(db *database.Manager) *Runner {
	return &Runner{
		db:      db,
		seeders: make([]Seeder, 0),
	}
}

// Register registers a seeder
func (r *Runner) Register(seeder Seeder) {
	r.seeders = append(r.seeders, seeder)
}

// Run runs all seeders
func (r *Runner) Run(ctx context.Context) error {
	for _, seeder := range r.seeders {
		if err := seeder.Run(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Example seeder implementation
type UserSeeder struct {
	userFactory *factories.UserFactory
}

func NewUserSeeder(userFactory *factories.UserFactory) *UserSeeder {
	return &UserSeeder{userFactory: userFactory}
}

func (s *UserSeeder) Run(ctx context.Context) error {
	// Create admin user
	_, err := s.userFactory.State("admin").Create(ctx, map[string]any{
		"name":  "Admin",
		"email": "admin@example.com",
	})
	if err != nil {
		return err
	}
	
	// Create regular users
	_, err = s.userFactory.CreateMany(ctx, 50)
	return err
}
```

## 4. HTTP Testing Utilities

```go
package testing

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	
	"github.com/yourusername/glib"
)

// TestRequest represents a test HTTP request
type TestRequest struct {
	app     *glib.Application
	method  string
	path    string
	headers map[string]string
	body    io.Reader
	t       *testing.T
}

// NewTestRequest creates a new test request
func NewTestRequest(t *testing.T, app *glib.Application) *TestRequest {
	return &TestRequest{
		app:     app,
		t:       t,
		headers: make(map[string]string),
	}
}

// Get performs a GET request
func (r *TestRequest) Get(path string) *TestResponse {
	return r.request("GET", path, nil)
}

// Post performs a POST request
func (r *TestRequest) Post(path string, body any) *TestResponse {
	return r.request("POST", path, body)
}

// Put performs a PUT request
func (r *TestRequest) Put(path string, body any) *TestResponse {
	return r.request("PUT", path, body)
}

// Delete performs a DELETE request
func (r *TestRequest) Delete(path string) *TestResponse {
	return r.request("DELETE", path, nil)
}

// WithHeader adds a header
func (r *TestRequest) WithHeader(key, value string) *TestRequest {
	r.headers[key] = value
	return r
}

// WithJSON sets Content-Type to application/json
func (r *TestRequest) WithJSON() *TestRequest {
	r.headers["Content-Type"] = "application/json"
	return r
}

// WithAuth sets Authorization header
func (r *TestRequest) WithAuth(token string) *TestRequest {
	r.headers["Authorization"] = "Bearer " + token
	return r
}

// request performs the actual request
func (r *TestRequest) request(method, path string, body any) *TestResponse {
	var bodyReader io.Reader
	
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			r.t.Fatal(err)
		}
		bodyReader = bytes.NewBuffer(jsonData)
	}
	
	req := httptest.NewRequest(method, path, bodyReader)
	
	// Set headers
	for key, value := range r.headers {
		req.Header.Set(key, value)
	}
	
	// Record response
	recorder := httptest.NewRecorder()
	
	// Serve request
	r.app.Router().ServeHTTP(recorder, req)
	
	return &TestResponse{
		t:        r.t,
		response: recorder.Result(),
		body:     recorder.Body.Bytes(),
	}
}

// TestResponse represents a test HTTP response
type TestResponse struct {
	t        *testing.T
	response *http.Response
	body     []byte
}

// AssertStatus asserts response status code
func (r *TestResponse) AssertStatus(expected int) *TestResponse {
	if r.response.StatusCode != expected {
		r.t.Errorf("expected status %d, got %d", expected, r.response.StatusCode)
	}
	return r
}

// AssertOK asserts 200 status
func (r *TestResponse) AssertOK() *TestResponse {
	return r.AssertStatus(http.StatusOK)
}

// AssertCreated asserts 201 status
func (r *TestResponse) AssertCreated() *TestResponse {
	return r.AssertStatus(http.StatusCreated)
}

// AssertUnauthorized asserts 401 status
func (r *TestResponse) AssertUnauthorized() *TestResponse {
	return r.AssertStatus(http.StatusUnauthorized)
}

// AssertNotFound asserts 404 status
func (r *TestResponse) AssertNotFound() *TestResponse {
	return r.AssertStatus(http.StatusNotFound)
}

// AssertJSON asserts response is JSON
func (r *TestResponse) AssertJSON() *TestResponse {
	contentType := r.response.Header.Get("Content-Type")
	if contentType != "application/json" && !strings.Contains(contentType, "application/json") {
		r.t.Errorf("expected JSON response, got %s", contentType)
	}
	return r
}

// AssertJSONPath asserts JSON path exists and has value
func (r *TestResponse) AssertJSONPath(path string, expected any) *TestResponse {
	var data map[string]any
	if err := json.Unmarshal(r.body, &data); err != nil {
		r.t.Fatal(err)
	}
	
	// Simple path navigation (e.g., "user.name")
	value := getJSONPath(data, path)
	
	if value != expected {
		r.t.Errorf("expected %v at path %s, got %v", expected, path, value)
	}
	
	return r
}

// AssertJSONStructure asserts JSON has expected structure
func (r *TestResponse) AssertJSONStructure(keys ...string) *TestResponse {
	var data map[string]any
	if err := json.Unmarshal(r.body, &data); err != nil {
		r.t.Fatal(err)
	}
	
	for _, key := range keys {
		if _, exists := data[key]; !exists {
			r.t.Errorf("expected key %s in JSON response", key)
		}
	}
	
	return r
}

// AssertHeader asserts header exists with value
func (r *TestResponse) AssertHeader(key, expected string) *TestResponse {
	actual := r.response.Header.Get(key)
	if actual != expected {
		r.t.Errorf("expected header %s to be %s, got %s", key, expected, actual)
	}
	return r
}

// JSON decodes response body as JSON
func (r *TestResponse) JSON(dest any) error {
	return json.Unmarshal(r.body, dest)
}

// Body returns response body
func (r *TestResponse) Body() []byte {
	return r.body
}
```

### HTTP Testing Example

```go
package handlers_test

import (
	"testing"
	
	"github.com/yourusername/glib/testing"
)

func TestUserAPI(t *testing.T) {
	app := setupTestApp(t)
	request := testing.NewTestRequest(t, app)
	
	// Test user creation
	response := request.
		WithJSON().
		Post("/api/users", map[string]any{
			"name":  "John Doe",
			"email": "john@example.com",
		})
	
	response.
		AssertCreated().
		AssertJSON().
		AssertJSONStructure("id", "name", "email", "created_at").
		AssertJSONPath("name", "John Doe")
	
	// Get user
	request.Get("/api/users/1").
		AssertOK().
		AssertJSONPath("id", 1)
	
	// Test authentication
	token := loginAndGetToken(t, app)
	
	request.
		WithAuth(token).
		Get("/api/profile").
		AssertOK()
	
	// Test unauthorized access
	request.Get("/api/profile").AssertUnauthorized()
}
```

## 5. Database Testing Helpers

```go
package testing

import (
	"context"
	"testing"
	
	"github.com/yourusername/glib/database"
)

// DatabaseTest provides database testing utilities
type DatabaseTest struct {
	db *database.Manager
	t  *testing.T
}

// NewDatabaseTest creates a new database test helper
func NewDatabaseTest(t *testing.T, db *database.Manager) *DatabaseTest {
	return &DatabaseTest{db: db, t: t}
}

// RefreshDatabase drops and recreates all tables
func (dt *DatabaseTest) RefreshDatabase(ctx context.Context) {
	conn, err := dt.db.Connection("")
	if err != nil {
		dt.t.Fatal(err)
	}
	
	// Drop all tables
	// Run migrations
	// Implementation depends on your migration system
}

// Seed runs database seeders
func (dt *DatabaseTest) Seed(ctx context.Context, seeders ...Seeder) {
	for _, seeder := range seeders {
		if err := seeder.Run(ctx); err != nil {
			dt.t.Fatal(err)
		}
	}
}

// AssertDatabaseHas asserts record exists in database
func (dt *DatabaseTest) AssertDatabaseHas(table string, conditions map[string]any) {
	conn, err := dt.db.Connection("")
	if err != nil {
		dt.t.Fatal(err)
	}
	
	query := conn.Table(table)
	for key, value := range conditions {
		query = query.Where(key+" = ?", value)
	}
	
	var count int64
	if err := query.Count(&count).Error; err != nil {
		dt.t.Fatal(err)
	}
	
	if count == 0 {
		dt.t.Errorf("expected database to have record in %s with %v", table, conditions)
	}
}

// AssertDatabaseMissing asserts record doesn't exist
func (dt *DatabaseTest) AssertDatabaseMissing(table string, conditions map[string]any) {
	conn, err := dt.db.Connection("")
	if err != nil {
		dt.t.Fatal(err)
	}
	
	query := conn.Table(table)
	for key, value := range conditions {
		query = query.Where(key+" = ?", value)
	}
	
	var count int64
	if err := query.Count(&count).Error; err != nil {
		dt.t.Fatal(err)
	}
	
	if count > 0 {
		dt.t.Errorf("expected database to not have record in %s with %v", table, conditions)
	}
}

// AssertDatabaseCount asserts table has specific record count
func (dt *DatabaseTest) AssertDatabaseCount(table string, expected int64) {
	conn, err := dt.db.Connection("")
	if err != nil {
		dt.t.Fatal(err)
	}
	
	var count int64
	if err := conn.Table(table).Count(&count).Error; err != nil {
		dt.t.Fatal(err)
	}
	
	if count != expected {
		dt.t.Errorf("expected %d records in %s, got %d", expected, table, count)
	}
}
```

## 6. Fake Implementations

```go
package fakes

import (
	"context"
	"sync"
	
	"github.com/yourusername/glib/cache"
	"github.com/yourusername/glib/storage"
	"github.com/yourusername/glib/queue"
)

// FakeCache implements a fake cache for testing
type FakeCache struct {
	items map[string]any
	mu    sync.RWMutex
}

func NewFakeCache() *FakeCache {
	return &FakeCache{
		items: make(map[string]any),
	}
}

func (c *FakeCache) AssertPut(key string, value any) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if _, exists := c.items[key]; !exists {
		panic(fmt.Sprintf("expected cache to have key %s", key))
	}
}

func (c *FakeCache) AssertMissing(key string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if _, exists := c.items[key]; exists {
		panic(fmt.Sprintf("expected cache to not have key %s", key))
	}
}

// FakeStorage implements a fake storage for testing
type FakeStorage struct {
	files map[string][]byte
	mu    sync.RWMutex
}

func NewFakeStorage() *FakeStorage {
	return &FakeStorage{
		files: make(map[string][]byte),
	}
}

func (s *FakeStorage) AssertExists(path string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if _, exists := s.files[path]; !exists {
		panic(fmt.Sprintf("expected storage to have file %s", path))
	}
}

func (s *FakeStorage) AssertMissing(path string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if _, exists := s.files[path]; exists {
		panic(fmt.Sprintf("expected storage to not have file %s", path))
	}
}

// FakeQueue implements a fake queue for testing
type FakeQueue struct {
	jobs []queue.Job
	mu   sync.RWMutex
}

func NewFakeQueue() *FakeQueue {
	return &FakeQueue{
		jobs: make([]queue.Job, 0),
	}
}

func (q *FakeQueue) AssertPushed(jobType string) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	
	for _, job := range q.jobs {
		if job.Name() == jobType {
			return
		}
	}
	
	panic(fmt.Sprintf("expected job %s to be pushed", jobType))
}

func (q *FakeQueue) AssertNotPushed(jobType string) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	
	for _, job := range q.jobs {
		if job.Name() == jobType {
			panic(fmt.Sprintf("expected job %s to not be pushed", jobType))
		}
	}
}

func (q *FakeQueue) AssertPushedCount(jobType string, expected int) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	
	count := 0
	for _, job := range q.jobs {
		if job.Name() == jobType {
			count++
		}
	}
	
	if count != expected {
		panic(fmt.Sprintf("expected %d jobs of type %s, got %d", expected, jobType, count))
	}
}
```

## 7. CLI Commands

```bash
# Generate factory
glib make:factory UserFactory

# Generate seeder
glib make:seeder UserSeeder

# Run seeders
glib db:seed                    # Run all seeders
glib db:seed --seeder=UserSeeder  # Run specific seeder

# Fresh database with seed
glib migrate:fresh --seed
```

## Success Criteria

1. **Collections API**
   - ✅ Generic collection type with type safety
   - ✅ Rich set of transformation methods
   - ✅ Laravel-style fluent API
   - ✅ Zero allocation for chainable operations where possible

2. **Model Factories**
   - ✅ Easy test data generation
   - ✅ State management for variants
   - ✅ Sequence support for unique values
   - ✅ Relationship handling

3. **Testing Utilities**
   - ✅ Fluent HTTP testing API
   - ✅ Database assertions
   - ✅ Fake implementations
   - ✅ Minimal boilerplate

4. **Developer Experience**
   - ✅ Intuitive, Laravel-like APIs
   - ✅ Type-safe operations
   - ✅ Excellent IDE support
   - ✅ Comprehensive examples

## Next Steps

Framework is now fully specified! Next actions:
1. Begin implementation starting with Phase 1 (Foundation)
2. Create additional supporting documentation
3. Set up CI/CD and testing infrastructure
4. Start building the community and documentation site
