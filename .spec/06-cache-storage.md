# Phase 6: Cache & File Storage

**Timeline**: Weeks 17-18  
**Priority**: High - Critical for performance and scalability  
**Dependencies**: Phase 1 (Foundation), Phase 2 (Database)

## Overview

Build a flexible caching layer and file storage abstraction inspired by Laravel's Cache and Storage facades with support for:

- Multiple cache drivers (In-Memory, Redis, Database, File)
- Cache tags for grouped invalidation
- Distributed cache locks
- Cache remember patterns
- File storage abstraction (Local, S3, GCS, Azure)
- Public and private file access
- Temporary signed URLs
- File streaming and chunking

## Package Structure

```
cache/
├── manager.go          # Cache manager
├── cache.go           # Cache interface
├── store.go           # Base store implementation
├── tagged_cache.go    # Tagged cache implementation
├── lock.go            # Cache lock interface
└── events.go          # Cache events

cache/drivers/
├── memory.go          # In-memory cache driver
├── redis.go           # Redis cache driver
├── database.go        # Database cache driver
└── file.go            # File-based cache driver

storage/
├── manager.go         # Storage manager
├── storage.go         # Storage interface
├── file.go            # File representation
├── visibility.go      # Public/private visibility
└── events.go          # Storage events

storage/drivers/
├── local.go           # Local filesystem driver
├── s3.go             # AWS S3 driver
├── gcs.go            # Google Cloud Storage driver
└── azure.go          # Azure Blob Storage driver
```

## 1. Cache System

### Cache Interface

```go
package cache

import (
 "context"
 "time"
)

// Cache represents a cache store
type Cache interface {
 // Get retrieves an item from cache
 Get(ctx context.Context, key string, dest any) error

 // Put stores an item in cache
 Put(ctx context.Context, key string, value any, ttl time.Duration) error

 // Forever stores an item forever
 Forever(ctx context.Context, key string, value any) error

 // Has checks if item exists in cache
 Has(ctx context.Context, key string) (bool, error)

 // Missing checks if item doesn't exist
 Missing(ctx context.Context, key string) (bool, error)

 // Increment increments a numeric value
 Increment(ctx context.Context, key string, value int64) (int64, error)

 // Decrement decrements a numeric value
 Decrement(ctx context.Context, key string, value int64) (int64, error)

 // Forget removes an item from cache
 Forget(ctx context.Context, key string) error

 // Flush clears all items from cache
 Flush(ctx context.Context) error

 // GetOrPut gets value or puts if missing (cache remember)
 GetOrPut(ctx context.Context, key string, dest any, ttl time.Duration, callback func() (any, error)) error

 // Remember gets value or stores callback result
 Remember(ctx context.Context, key string, dest any, ttl time.Duration, callback func() (any, error)) error

 // RememberForever gets value or stores callback result forever
 RememberForever(ctx context.Context, key string, dest any, callback func() (any, error)) error

 // Pull retrieves and deletes an item
 Pull(ctx context.Context, key string, dest any) error

 // Add stores item only if it doesn't exist
 Add(ctx context.Context, key string, value any, ttl time.Duration) (bool, error)

 // Tags returns a tagged cache instance
 Tags(names ...string) TaggedCache

 // Lock creates a cache lock
 Lock(name string, ttl time.Duration) Lock
}

// TaggedCache represents a cache with tags
type TaggedCache interface {
 Cache

 // FlushTags flushes all items with these tags
 FlushTags(ctx context.Context) error
}

// Lock represents a distributed lock
type Lock interface {
 // Acquire attempts to acquire the lock
 Acquire(ctx context.Context) (bool, error)

 // Release releases the lock
 Release(ctx context.Context) error

 // ForceRelease forcefully releases the lock
 ForceRelease(ctx context.Context) error

 // Owner returns the lock owner identifier
 Owner() string

 // Get executes callback while holding lock
 Get(ctx context.Context, callback func() error) error

 // Block waits to acquire lock then executes callback
 Block(ctx context.Context, waitTime time.Duration, callback func() error) error
}
```

### Cache Manager

```go
package cache

import (
 "fmt"
 "sync"

 "github.com/yourusername/glib/config"
 "github.com/yourusername/glib/container"
)

// Manager manages cache stores
type Manager struct {
 app    *container.Container
 config config.Config
 stores map[string]Cache
 mu     sync.RWMutex
}

// NewManager creates a new cache manager
func NewManager(app *container.Container, cfg config.Config) *Manager {
 return &Manager{
  app:    app,
  config: cfg,
  stores: make(map[string]Cache),
 }
}

// Store returns a cache store by name
func (m *Manager) Store(name string) (Cache, error) {
 if name == "" {
  name = m.config.GetString("cache.default", "memory")
 }

 m.mu.RLock()
 store, exists := m.stores[name]
 m.mu.RUnlock()

 if exists {
  return store, nil
 }

 return m.resolve(name)
}

// resolve creates a new cache store
func (m *Manager) resolve(name string) (Cache, error) {
 m.mu.Lock()
 defer m.mu.Unlock()

 // Double-check after acquiring write lock
 if store, exists := m.stores[name]; exists {
  return store, nil
 }

 driver := m.config.GetString(fmt.Sprintf("cache.stores.%s.driver", name))

 var store Cache
 var err error

 switch driver {
 case "memory":
  store = NewMemoryStore()
 case "redis":
  store, err = m.createRedisStore(name)
 case "database":
  store, err = m.createDatabaseStore(name)
 case "file":
  store, err = m.createFileStore(name)
 default:
  return nil, fmt.Errorf("unsupported cache driver: %s", driver)
 }

 if err != nil {
  return nil, err
 }

 m.stores[name] = store
 return store, nil
}

// Default returns the default cache store
func (m *Manager) Default() (Cache, error) {
 return m.Store("")
}
```

### Memory Cache Driver

```go
package drivers

import (
 "context"
 "sync"
 "time"

 "github.com/yourusername/glib/cache"
)

type item struct {
 value      any
 expiration time.Time
 forever    bool
}

// MemoryStore implements in-memory caching
type MemoryStore struct {
 items map[string]*item
 mu    sync.RWMutex
}

// NewMemoryStore creates a new memory store
func NewMemoryStore() *MemoryStore {
 store := &MemoryStore{
  items: make(map[string]*item),
 }

 // Start cleanup goroutine
 go store.cleanup()

 return store
}

func (s *MemoryStore) Get(ctx context.Context, key string, dest any) error {
 s.mu.RLock()
 defer s.mu.RUnlock()

 item, exists := s.items[key]
 if !exists {
  return cache.ErrCacheMiss
 }

 if !item.forever && time.Now().After(item.expiration) {
  return cache.ErrCacheMiss
 }

 // Use reflection or type assertion to copy value
 return copyValue(item.value, dest)
}

func (s *MemoryStore) Put(ctx context.Context, key string, value any, ttl time.Duration) error {
 s.mu.Lock()
 defer s.mu.Unlock()

 s.items[key] = &item{
  value:      value,
  expiration: time.Now().Add(ttl),
  forever:    false,
 }

 return nil
}

func (s *MemoryStore) Forever(ctx context.Context, key string, value any) error {
 s.mu.Lock()
 defer s.mu.Unlock()

 s.items[key] = &item{
  value:   value,
  forever: true,
 }

 return nil
}

func (s *MemoryStore) Has(ctx context.Context, key string) (bool, error) {
 s.mu.RLock()
 defer s.mu.RUnlock()

 item, exists := s.items[key]
 if !exists {
  return false, nil
 }

 if !item.forever && time.Now().After(item.expiration) {
  return false, nil
 }

 return true, nil
}

func (s *MemoryStore) Forget(ctx context.Context, key string) error {
 s.mu.Lock()
 defer s.mu.Unlock()

 delete(s.items, key)
 return nil
}

func (s *MemoryStore) Flush(ctx context.Context) error {
 s.mu.Lock()
 defer s.mu.Unlock()

 s.items = make(map[string]*item)
 return nil
}

func (s *MemoryStore) cleanup() {
 ticker := time.NewTicker(1 * time.Minute)
 defer ticker.Stop()

 for range ticker.C {
  s.mu.Lock()
  now := time.Now()
  for key, item := range s.items {
   if !item.forever && now.After(item.expiration) {
    delete(s.items, key)
   }
  }
  s.mu.Unlock()
 }
}
```

### Redis Cache Driver

```go
package drivers

import (
 "context"
 "encoding/json"
 "time"

 "github.com/redis/go-redis/v9"
 "github.com/yourusername/glib/cache"
)

// RedisStore implements Redis caching
type RedisStore struct {
 client *redis.Client
 prefix string
}

// NewRedisStore creates a new Redis store
func NewRedisStore(client *redis.Client, prefix string) *RedisStore {
 return &RedisStore{
  client: client,
  prefix: prefix,
 }
}

func (s *RedisStore) Get(ctx context.Context, key string, dest any) error {
 data, err := s.client.Get(ctx, s.prefix+key).Bytes()
 if err == redis.Nil {
  return cache.ErrCacheMiss
 }
 if err != nil {
  return err
 }

 return json.Unmarshal(data, dest)
}

func (s *RedisStore) Put(ctx context.Context, key string, value any, ttl time.Duration) error {
 data, err := json.Marshal(value)
 if err != nil {
  return err
 }

 return s.client.Set(ctx, s.prefix+key, data, ttl).Err()
}

func (s *RedisStore) Forever(ctx context.Context, key string, value any) error {
 data, err := json.Marshal(value)
 if err != nil {
  return err
 }

 return s.client.Set(ctx, s.prefix+key, data, 0).Err()
}

func (s *RedisStore) Increment(ctx context.Context, key string, value int64) (int64, error) {
 return s.client.IncrBy(ctx, s.prefix+key, value).Result()
}

func (s *RedisStore) Decrement(ctx context.Context, key string, value int64) (int64, error) {
 return s.client.DecrBy(ctx, s.prefix+key, value).Result()
}

func (s *RedisStore) Forget(ctx context.Context, key string) error {
 return s.client.Del(ctx, s.prefix+key).Err()
}

func (s *RedisStore) Flush(ctx context.Context) error {
 return s.client.FlushDB(ctx).Err()
}

// Tags returns a tagged cache instance for Redis
func (s *RedisStore) Tags(names ...string) cache.TaggedCache {
 return NewRedisTaggedCache(s, names)
}

// Lock creates a Redis-based distributed lock
func (s *RedisStore) Lock(name string, ttl time.Duration) cache.Lock {
 return NewRedisLock(s.client, s.prefix+"lock:"+name, ttl)
}
```

### Cache Lock Implementation

```go
package cache

import (
 "context"
 "crypto/rand"
 "encoding/hex"
 "time"

 "github.com/redis/go-redis/v9"
)

// RedisLock implements distributed locking with Redis
type RedisLock struct {
 client *redis.Client
 name   string
 owner  string
 ttl    time.Duration
}

// NewRedisLock creates a new Redis lock
func NewRedisLock(client *redis.Client, name string, ttl time.Duration) *RedisLock {
 return &RedisLock{
  client: client,
  name:   name,
  owner:  generateOwnerID(),
  ttl:    ttl,
 }
}

func (l *RedisLock) Acquire(ctx context.Context) (bool, error) {
 success, err := l.client.SetNX(ctx, l.name, l.owner, l.ttl).Result()
 return success, err
}

func (l *RedisLock) Release(ctx context.Context) error {
 script := `
  if redis.call("get", KEYS[1]) == ARGV[1] then
   return redis.call("del", KEYS[1])
  else
   return 0
  end
 `
 return l.client.Eval(ctx, script, []string{l.name}, l.owner).Err()
}

func (l *RedisLock) ForceRelease(ctx context.Context) error {
 return l.client.Del(ctx, l.name).Err()
}

func (l *RedisLock) Owner() string {
 return l.owner
}

func (l *RedisLock) Get(ctx context.Context, callback func() error) error {
 acquired, err := l.Acquire(ctx)
 if err != nil {
  return err
 }
 if !acquired {
  return ErrLockNotAcquired
 }

 defer l.Release(ctx)

 return callback()
}

func (l *RedisLock) Block(ctx context.Context, waitTime time.Duration, callback func() error) error {
 deadline := time.Now().Add(waitTime)

 for {
  acquired, err := l.Acquire(ctx)
  if err != nil {
   return err
  }

  if acquired {
   defer l.Release(ctx)
   return callback()
  }

  if time.Now().After(deadline) {
   return ErrLockTimeout
  }

  // Wait a bit before retrying
  time.Sleep(100 * time.Millisecond)
 }
}

func generateOwnerID() string {
 bytes := make([]byte, 16)
 rand.Read(bytes)
 return hex.EncodeToString(bytes)
}
```

### Cache Helper Functions

```go
package cache

// BaseStore provides common cache functionality
type BaseStore struct {
 Cache
}

func (s *BaseStore) Remember(ctx context.Context, key string, dest any, ttl time.Duration, callback func() (any, error)) error {
 // Try to get from cache
 err := s.Get(ctx, key, dest)
 if err == nil {
  return nil // Cache hit
 }

 if err != ErrCacheMiss {
  return err // Real error
 }

 // Cache miss - execute callback
 value, err := callback()
 if err != nil {
  return err
 }

 // Store in cache
 if err := s.Put(ctx, key, value, ttl); err != nil {
  return err
 }

 // Copy value to dest
 return copyValue(value, dest)
}

func (s *BaseStore) RememberForever(ctx context.Context, key string, dest any, callback func() (any, error)) error {
 err := s.Get(ctx, key, dest)
 if err == nil {
  return nil
 }

 if err != ErrCacheMiss {
  return err
 }

 value, err := callback()
 if err != nil {
  return err
 }

 if err := s.Forever(ctx, key, value); err != nil {
  return err
 }

 return copyValue(value, dest)
}

func (s *BaseStore) Pull(ctx context.Context, key string, dest any) error {
 err := s.Get(ctx, key, dest)
 if err != nil {
  return err
 }

 return s.Forget(ctx, key)
}

func (s *BaseStore) Add(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
 has, err := s.Has(ctx, key)
 if err != nil {
  return false, err
 }

 if has {
  return false, nil
 }

 return true, s.Put(ctx, key, value, ttl)
}
```

## 2. File Storage System

### Storage Interface

```go
package storage

import (
 "context"
 "io"
 "time"
)

// Storage represents a file storage driver
type Storage interface {
 // Put stores a file
 Put(ctx context.Context, path string, contents io.Reader) error

 // PutFile stores an uploaded file
 PutFile(ctx context.Context, path string, file *File) (string, error)

 // Get retrieves file contents
 Get(ctx context.Context, path string) ([]byte, error)

 // GetStream retrieves file as a stream
 GetStream(ctx context.Context, path string) (io.ReadCloser, error)

 // Exists checks if file exists
 Exists(ctx context.Context, path string) (bool, error)

 // Missing checks if file doesn't exist
 Missing(ctx context.Context, path string) (bool, error)

 // Size returns file size in bytes
 Size(ctx context.Context, path string) (int64, error)

 // LastModified returns file modification time
 LastModified(ctx context.Context, path string) (time.Time, error)

 // Delete removes a file
 Delete(ctx context.Context, path string) error

 // DeleteDirectory removes a directory
 DeleteDirectory(ctx context.Context, path string) error

 // Copy copies a file
 Copy(ctx context.Context, from, to string) error

 // Move moves a file
 Move(ctx context.Context, from, to string) error

 // Files lists all files in a directory
 Files(ctx context.Context, directory string) ([]string, error)

 // AllFiles lists all files recursively
 AllFiles(ctx context.Context, directory string) ([]string, error)

 // Directories lists all directories
 Directories(ctx context.Context, directory string) ([]string, error)

 // AllDirectories lists all directories recursively
 AllDirectories(ctx context.Context, directory string) ([]string, error)

 // MakeDirectory creates a directory
 MakeDirectory(ctx context.Context, path string) error

 // URL gets the URL for a file
 URL(ctx context.Context, path string) (string, error)

 // TemporaryURL generates a temporary URL
 TemporaryURL(ctx context.Context, path string, expiration time.Time) (string, error)

 // SetVisibility sets file visibility (public/private)
 SetVisibility(ctx context.Context, path string, visibility Visibility) error

 // GetVisibility gets file visibility
 GetVisibility(ctx context.Context, path string) (Visibility, error)
}

// Visibility represents file visibility
type Visibility string

const (
 VisibilityPublic  Visibility = "public"
 VisibilityPrivate Visibility = "private"
)

// File represents an uploaded file
type File struct {
 Name        string
 Size        int64
 MimeType    string
 Extension   string
 Reader      io.Reader
 Visibility  Visibility
}
```

### Storage Manager

```go
package storage

import (
 "fmt"
 "sync"

 "github.com/yourusername/glib/config"
 "github.com/yourusername/glib/container"
)

// Manager manages storage disks
type Manager struct {
 app    *container.Container
 config config.Config
 disks  map[string]Storage
 mu     sync.RWMutex
}

// NewManager creates a new storage manager
func NewManager(app *container.Container, cfg config.Config) *Manager {
 return &Manager{
  app:    app,
  config: cfg,
  disks:  make(map[string]Storage),
 }
}

// Disk returns a storage disk by name
func (m *Manager) Disk(name string) (Storage, error) {
 if name == "" {
  name = m.config.GetString("storage.default", "local")
 }

 m.mu.RLock()
 disk, exists := m.disks[name]
 m.mu.RUnlock()

 if exists {
  return disk, nil
 }

 return m.resolve(name)
}

// resolve creates a new storage disk
func (m *Manager) resolve(name string) (Storage, error) {
 m.mu.Lock()
 defer m.mu.Unlock()

 // Double-check after acquiring write lock
 if disk, exists := m.disks[name]; exists {
  return disk, nil
 }

 driver := m.config.GetString(fmt.Sprintf("storage.disks.%s.driver", name))

 var disk Storage
 var err error

 switch driver {
 case "local":
  disk, err = m.createLocalDisk(name)
 case "s3":
  disk, err = m.createS3Disk(name)
 case "gcs":
  disk, err = m.createGCSDisk(name)
 case "azure":
  disk, err = m.createAzureDisk(name)
 default:
  return nil, fmt.Errorf("unsupported storage driver: %s", driver)
 }

 if err != nil {
  return nil, err
 }

 m.disks[name] = disk
 return disk, nil
}

// Default returns the default storage disk
func (m *Manager) Default() (Storage, error) {
 return m.Disk("")
}
```

### Local Storage Driver

```go
package drivers

import (
 "context"
 "fmt"
 "io"
 "os"
 "path/filepath"
 "time"

 "github.com/yourusername/glib/storage"
)

// LocalStorage implements local filesystem storage
type LocalStorage struct {
 root       string
 urlBase    string
 visibility storage.Visibility
}

// NewLocalStorage creates a new local storage driver
func NewLocalStorage(root, urlBase string, visibility storage.Visibility) (*LocalStorage, error) {
 // Ensure root directory exists
 if err := os.MkdirAll(root, 0755); err != nil {
  return nil, err
 }

 return &LocalStorage{
  root:       root,
  urlBase:    urlBase,
  visibility: visibility,
 }, nil
}

func (s *LocalStorage) Put(ctx context.Context, path string, contents io.Reader) error {
 fullPath := s.fullPath(path)

 // Ensure directory exists
 if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
  return err
 }

 file, err := os.Create(fullPath)
 if err != nil {
  return err
 }
 defer file.Close()

 _, err = io.Copy(file, contents)
 return err
}

func (s *LocalStorage) Get(ctx context.Context, path string) ([]byte, error) {
 return os.ReadFile(s.fullPath(path))
}

func (s *LocalStorage) GetStream(ctx context.Context, path string) (io.ReadCloser, error) {
 return os.Open(s.fullPath(path))
}

func (s *LocalStorage) Exists(ctx context.Context, path string) (bool, error) {
 _, err := os.Stat(s.fullPath(path))
 if err == nil {
  return true, nil
 }
 if os.IsNotExist(err) {
  return false, nil
 }
 return false, err
}

func (s *LocalStorage) Size(ctx context.Context, path string) (int64, error) {
 info, err := os.Stat(s.fullPath(path))
 if err != nil {
  return 0, err
 }
 return info.Size(), nil
}

func (s *LocalStorage) LastModified(ctx context.Context, path string) (time.Time, error) {
 info, err := os.Stat(s.fullPath(path))
 if err != nil {
  return time.Time{}, err
 }
 return info.ModTime(), nil
}

func (s *LocalStorage) Delete(ctx context.Context, path string) error {
 return os.Remove(s.fullPath(path))
}

func (s *LocalStorage) DeleteDirectory(ctx context.Context, path string) error {
 return os.RemoveAll(s.fullPath(path))
}

func (s *LocalStorage) Copy(ctx context.Context, from, to string) error {
 source, err := os.Open(s.fullPath(from))
 if err != nil {
  return err
 }
 defer source.Close()

 // Ensure destination directory exists
 destPath := s.fullPath(to)
 if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
  return err
 }

 dest, err := os.Create(destPath)
 if err != nil {
  return err
 }
 defer dest.Close()

 _, err = io.Copy(dest, source)
 return err
}

func (s *LocalStorage) Move(ctx context.Context, from, to string) error {
 return os.Rename(s.fullPath(from), s.fullPath(to))
}

func (s *LocalStorage) Files(ctx context.Context, directory string) ([]string, error) {
 fullPath := s.fullPath(directory)

 entries, err := os.ReadDir(fullPath)
 if err != nil {
  return nil, err
 }

 var files []string
 for _, entry := range entries {
  if !entry.IsDir() {
   files = append(files, filepath.Join(directory, entry.Name()))
  }
 }

 return files, nil
}

func (s *LocalStorage) AllFiles(ctx context.Context, directory string) ([]string, error) {
 var files []string

 err := filepath.Walk(s.fullPath(directory), func(path string, info os.FileInfo, err error) error {
  if err != nil {
   return err
  }

  if !info.IsDir() {
   relPath, err := filepath.Rel(s.root, path)
   if err != nil {
    return err
   }
   files = append(files, relPath)
  }

  return nil
 })

 return files, err
}

func (s *LocalStorage) MakeDirectory(ctx context.Context, path string) error {
 return os.MkdirAll(s.fullPath(path), 0755)
}

func (s *LocalStorage) URL(ctx context.Context, path string) (string, error) {
 return fmt.Sprintf("%s/%s", s.urlBase, path), nil
}

func (s *LocalStorage) TemporaryURL(ctx context.Context, path string, expiration time.Time) (string, error) {
 // Local storage doesn't support signed URLs
 // Return regular URL or implement custom signed URL mechanism
 return s.URL(ctx, path)
}

func (s *LocalStorage) fullPath(path string) string {
 return filepath.Join(s.root, path)
}
```

### S3 Storage Driver

```go
package drivers

import (
 "bytes"
 "context"
 "fmt"
 "io"
 "time"

 "github.com/aws/aws-sdk-go-v2/aws"
 "github.com/aws/aws-sdk-go-v2/service/s3"
 "github.com/yourusername/glib/storage"
)

// S3Storage implements AWS S3 storage
type S3Storage struct {
 client     *s3.Client
 bucket     string
 region     string
 urlBase    string
 visibility storage.Visibility
}

// NewS3Storage creates a new S3 storage driver
func NewS3Storage(client *s3.Client, bucket, region, urlBase string, visibility storage.Visibility) *S3Storage {
 return &S3Storage{
  client:     client,
  bucket:     bucket,
  region:     region,
  urlBase:    urlBase,
  visibility: visibility,
 }
}

func (s *S3Storage) Put(ctx context.Context, path string, contents io.Reader) error {
 // Read all contents into buffer
 data, err := io.ReadAll(contents)
 if err != nil {
  return err
 }

 _, err = s.client.PutObject(ctx, &s3.PutObjectInput{
  Bucket: aws.String(s.bucket),
  Key:    aws.String(path),
  Body:   bytes.NewReader(data),
  ACL:    s.getACL(),
 })

 return err
}

func (s *S3Storage) Get(ctx context.Context, path string) ([]byte, error) {
 result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
  Bucket: aws.String(s.bucket),
  Key:    aws.String(path),
 })
 if err != nil {
  return nil, err
 }
 defer result.Body.Close()

 return io.ReadAll(result.Body)
}

func (s *S3Storage) GetStream(ctx context.Context, path string) (io.ReadCloser, error) {
 result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
  Bucket: aws.String(s.bucket),
  Key:    aws.String(path),
 })
 if err != nil {
  return nil, err
 }

 return result.Body, nil
}

func (s *S3Storage) Exists(ctx context.Context, path string) (bool, error) {
 _, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
  Bucket: aws.String(s.bucket),
  Key:    aws.String(path),
 })

 if err != nil {
  // Check if error is "not found"
  return false, nil
 }

 return true, nil
}

func (s *S3Storage) Delete(ctx context.Context, path string) error {
 _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
  Bucket: aws.String(s.bucket),
  Key:    aws.String(path),
 })

 return err
}

func (s *S3Storage) URL(ctx context.Context, path string) (string, error) {
 if s.urlBase != "" {
  return fmt.Sprintf("%s/%s", s.urlBase, path), nil
 }

 return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, path), nil
}

func (s *S3Storage) TemporaryURL(ctx context.Context, path string, expiration time.Time) (string, error) {
 presignClient := s3.NewPresignClient(s.client)

 request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
  Bucket: aws.String(s.bucket),
  Key:    aws.String(path),
 }, func(opts *s3.PresignOptions) {
  opts.Expires = time.Until(expiration)
 })

 if err != nil {
  return "", err
 }

 return request.URL, nil
}

func (s *S3Storage) getACL() types.ObjectCannedACL {
 if s.visibility == storage.VisibilityPublic {
  return types.ObjectCannedACLPublicRead
 }
 return types.ObjectCannedACLPrivate
}
```

## 3. Configuration

```yaml
# config/cache.yaml
cache:
  default: redis

  stores:
    memory:
      driver: memory

    redis:
      driver: redis
      connection: default
      prefix: cache_

    database:
      driver: database
      table: cache
      connection: default

    file:
      driver: file
      path: storage/cache

# config/storage.yaml
storage:
  default: local

  disks:
    local:
      driver: local
      root: storage/app
      url: http://localhost:8080/storage
      visibility: private

    public:
      driver: local
      root: storage/app/public
      url: http://localhost:8080
      visibility: public

    s3:
      driver: s3
      region: us-east-1
      bucket: my-bucket
      url: https://cdn.example.com
      visibility: public

    gcs:
      driver: gcs
      bucket: my-bucket
      project: my-project
      visibility: public
```

## 4. Usage Examples

### Cache Usage

```go
package main

import (
 "context"
 "time"

 "github.com/yourusername/glib/cache"
)

func HandleRequest(cacheManager *cache.Manager) error {
 ctx := context.Background()

 // Get default cache store
 cache, err := cacheManager.Default()
 if err != nil {
  return err
 }

 // Simple cache operations
 cache.Put(ctx, "user:1", user, 10*time.Minute)

 var user User
 err = cache.Get(ctx, "user:1", &user)

 // Cache remember pattern
 var posts []Post
 err = cache.Remember(ctx, "posts:recent", &posts, 5*time.Minute, func() (any, error) {
  return db.Query("SELECT * FROM posts ORDER BY created_at DESC LIMIT 10").Get()
 })

 // Cache with tags
 taggedCache := cache.Tags("users", "posts")
 taggedCache.Put(ctx, "user:1:posts", userPosts, 10*time.Minute)

 // Invalidate all cached items with these tags
 taggedCache.FlushTags(ctx)

 // Distributed lock
 lock := cache.Lock("process-order:123", 10*time.Second)

 lock.Get(ctx, func() error {
  // Critical section - only one process can execute this
  return processOrder(123)
 })

 // Block and wait for lock
 lock.Block(ctx, 30*time.Second, func() error {
  return processOrder(123)
 })

 return nil
}
```

### Storage Usage

```go
package main

import (
 "context"
 "bytes"
 "time"

 "github.com/yourusername/glib/storage"
)

func HandleFileUpload(storageManager *storage.Manager, file *storage.File) error {
 ctx := context.Background()

 // Get storage disk
 disk, err := storageManager.Disk("s3")
 if err != nil {
  return err
 }

 // Store file
 path, err := disk.PutFile(ctx, "uploads", file)
 if err != nil {
  return err
 }

 // Get file URL
 url, err := disk.URL(ctx, path)

 // Generate temporary URL (expires in 1 hour)
 tempURL, err := disk.TemporaryURL(ctx, path, time.Now().Add(1*time.Hour))

 // Check if file exists
 exists, err := disk.Exists(ctx, path)

 // Get file size
 size, err := disk.Size(ctx, path)

 // Copy file
 err = disk.Copy(ctx, path, "backups/"+path)

 // Move file
 err = disk.Move(ctx, path, "archive/"+path)

 // Delete file
 err = disk.Delete(ctx, path)

 // Stream file
 stream, err := disk.GetStream(ctx, path)
 defer stream.Close()

 // List files in directory
 files, err := disk.Files(ctx, "uploads")

 // List all files recursively
 allFiles, err := disk.AllFiles(ctx, "uploads")

 return nil
}

// In HTTP handler
func UploadAvatar(c *glib.Context, storage *storage.Manager) error {
 file, err := c.FormFile("avatar")
 if err != nil {
  return err
 }

 disk, _ := storage.Disk("public")

 path, err := disk.PutFile(c.Request.Context(), "avatars", &storage.File{
  Name:       file.Filename,
  Size:       file.Size,
  Reader:     file,
  Visibility: storage.VisibilityPublic,
 })

 if err != nil {
  return err
 }

 url, _ := disk.URL(c.Request.Context(), path)

 return c.JSON(200, map[string]any{
  "path": path,
  "url":  url,
 })
}
```

## 5. Service Provider

```go
package providers

import (
 "github.com/yourusername/glib/cache"
 "github.com/yourusername/glib/container"
 "github.com/yourusername/glib/storage"
)

// CacheServiceProvider registers cache services
type CacheServiceProvider struct{}

func (p *CacheServiceProvider) Register(app *container.Container) error {
 return app.Singleton(func(app *container.Container) (*cache.Manager, error) {
  cfg := app.MustResolve((*config.Config)(nil)).(config.Config)
  return cache.NewManager(app, cfg), nil
 })
}

func (p *CacheServiceProvider) Boot(app *container.Container) error {
 return nil
}

// StorageServiceProvider registers storage services
type StorageServiceProvider struct{}

func (p *StorageServiceProvider) Register(app *container.Container) error {
 return app.Singleton(func(app *container.Container) (*storage.Manager, error) {
  cfg := app.MustResolve((*config.Config)(nil)).(config.Config)
  return storage.NewManager(app, cfg), nil
 })
}

func (p *StorageServiceProvider) Boot(app *container.Container) error {
 return nil
}
```

## 6. Testing Support

```go
package cache_test

import (
 "context"
 "testing"
 "time"

 "github.com/yourusername/glib/cache/drivers"
)

func TestMemoryCache(t *testing.T) {
 cache := drivers.NewMemoryStore()
 ctx := context.Background()

 // Test Put and Get
 err := cache.Put(ctx, "key", "value", 1*time.Minute)
 if err != nil {
  t.Fatal(err)
 }

 var result string
 err = cache.Get(ctx, "key", &result)
 if err != nil {
  t.Fatal(err)
 }

 if result != "value" {
  t.Errorf("expected 'value', got '%s'", result)
 }

 // Test expiration
 cache.Put(ctx, "temp", "data", 100*time.Millisecond)
 time.Sleep(200 * time.Millisecond)

 err = cache.Get(ctx, "temp", &result)
 if err != cache.ErrCacheMiss {
  t.Error("expected cache miss for expired item")
 }

 // Test Remember
 called := false
 err = cache.Remember(ctx, "computed", &result, 1*time.Minute, func() (any, error) {
  called = true
  return "computed value", nil
 })

 if !called {
  t.Error("callback should have been called")
 }

 // Second call should use cache
 called = false
 cache.Remember(ctx, "computed", &result, 1*time.Minute, func() (any, error) {
  called = true
  return "new value", nil
 })

 if called {
  t.Error("callback should not have been called on cache hit")
 }
}
```

## 7. CLI Commands

```bash
# Cache commands
glib cache:clear              # Clear all caches
glib cache:clear redis        # Clear specific cache store
glib cache:forget user:123    # Remove specific cache key
glib cache:table              # Create cache database table migration

# Storage commands
glib storage:link             # Create symbolic link for public storage
glib storage:clean            # Clean up old temporary files
```

## Success Criteria

1. **Cache System**
   - ✅ Multiple drivers (Memory, Redis, Database, File)
   - ✅ Cache remember pattern
   - ✅ Tagged caching for grouped invalidation
   - ✅ Distributed locks with Redis
   - ✅ Atomic increment/decrement operations

2. **File Storage**
   - ✅ Multiple drivers (Local, S3, GCS, Azure)
   - ✅ Unified interface across all drivers
   - ✅ Temporary signed URLs
   - ✅ Public/private visibility
   - ✅ File streaming support

3. **Developer Experience**
   - ✅ Simple, intuitive API
   - ✅ Laravel-style elegance
   - ✅ Type-safe operations
   - ✅ Comprehensive error handling

4. **Performance**
   - ✅ Efficient caching with minimal overhead
   - ✅ Connection pooling for Redis
   - ✅ Streaming for large files
   - ✅ No unnecessary allocations

## Implementation Notes

1. **Cache Serialization**
   - Use JSON for cross-language compatibility
   - Consider msgpack for better performance
   - Support custom serializers

2. **Storage Chunking**
   - Implement multipart uploads for large files
   - Support resumable uploads
   - Progress callbacks for uploads

3. **Error Handling**
   - Distinguish between cache miss and errors
   - Retry logic for network failures
   - Circuit breakers for external services

4. **Testing**
   - Mock storage drivers for testing
   - In-memory cache for integration tests
   - Test helpers for common scenarios

## Next Steps

After completing Phase 6, proceed to:

- **Phase 7**: Developer Experience (Collections, Factories, Testing Utilities)
- Additional features: Events, Broadcasting, Mail
