package scanner

import (
	"fmt"
	"go/ast"
	"go/parser"
	"os"
	"runtime"
	"sync"
	"time"
)

// WorkerPool manages parallel file parsing
type WorkerPool struct {
	workers int
	jobs    chan *parseJob
	results chan *ParseResult
	errors  chan error
	wg      sync.WaitGroup
	scanner *Scanner
}

// parseJob represents a file to be parsed
type parseJob struct {
	path       string
	info       os.FileInfo
	fromCache  bool
	cacheEntry *CacheEntry
}

// ParseResult contains the parsed results from a file
type ParseResult struct {
	FilePath    string
	File        *ast.File // Only populated if needed for handlers pass
	Controllers []*Controller
	Providers   []*Provider
	Middleware  []*Middleware
	Configs     []*Config
	Hash        string
	ModTime     time.Time
}

// NewWorkerPool creates a new worker pool
// If workers <= 0, uses runtime.NumCPU() / 2
func NewWorkerPool(scanner *Scanner, workers int) *WorkerPool {
	if workers <= 0 {
		workers = runtime.NumCPU() / 2
		if workers < 1 {
			workers = 1
		}
	}

	return &WorkerPool{
		workers: workers,
		jobs:    make(chan *parseJob, workers*2), // Buffer for better throughput
		results: make(chan *ParseResult, workers*2),
		errors:  make(chan error, workers),
		scanner: scanner,
	}
}

// Start starts the worker pool
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}
}

// Submit submits a job to the worker pool
func (wp *WorkerPool) Submit(path string, info os.FileInfo) {
	wp.jobs <- &parseJob{
		path: path,
		info: info,
	}
}

// SubmitCached submits a cached result
func (wp *WorkerPool) SubmitCached(path string, entry *CacheEntry) {
	wp.jobs <- &parseJob{
		path:       path,
		fromCache:  true,
		cacheEntry: entry,
	}
}

// Close closes the job channel and waits for workers to finish
func (wp *WorkerPool) Close() {
	close(wp.jobs)
	wp.wg.Wait()
	close(wp.results)
	close(wp.errors)
}

// Results returns the results channel
func (wp *WorkerPool) Results() <-chan *ParseResult {
	return wp.results
}

// Errors returns the errors channel
func (wp *WorkerPool) Errors() <-chan error {
	return wp.errors
}

// worker processes jobs from the queue
func (wp *WorkerPool) worker() {
	defer wp.wg.Done()

	for job := range wp.jobs {
		// Check if using cached result
		if job.fromCache && job.cacheEntry != nil {
			wp.results <- &ParseResult{
				FilePath:    job.path,
				Controllers: job.cacheEntry.Controllers,
				Providers:   job.cacheEntry.Providers,
				Middleware:  job.cacheEntry.Middleware,
				Configs:     job.cacheEntry.Configs,
				Hash:        job.cacheEntry.Hash,
				ModTime:     job.cacheEntry.ModTime,
			}
			continue
		}

		// Parse and process the file
		result, err := wp.processFile(job)
		if err != nil {
			wp.errors <- fmt.Errorf("failed to process %s: %w", job.path, err)
			continue
		}

		wp.results <- result
	}
}

// processFile parses and scans a single file
func (wp *WorkerPool) processFile(job *parseJob) (*ParseResult, error) {
	// Parse the file
	file, err := parser.ParseFile(wp.scanner.fset, job.path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// Create temporary project to collect results
	tempProject := &Project{
		Module: wp.scanner.modulePath,
	}

	// Scan the file
	if err := wp.scanner.scanFile(file, job.path, tempProject); err != nil {
		return nil, err
	}

	// Compute hash for caching
	hash := ""
	if wp.scanner.cacheEnabled {
		hash, _ = computeFileHash(job.path)
	}

	result := &ParseResult{
		FilePath:    job.path,
		File:        file,
		Controllers: tempProject.Controllers,
		Providers:   tempProject.Providers,
		Middleware:  tempProject.Middleware,
		Configs:     tempProject.Configs,
		Hash:        hash,
		ModTime:     job.info.ModTime(),
	}

	return result, nil
}

// WorkerCount returns the number of workers
func (wp *WorkerPool) WorkerCount() int {
	return wp.workers
}
