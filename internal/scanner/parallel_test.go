package scanner

import (
	"runtime"
	"testing"
)

func TestWorkerPool(t *testing.T) {
	projectDir := "../../examples/demo"

	scanner, err := New(projectDir)
	if err != nil {
		t.Fatalf("failed to create scanner: %v", err)
	}

	t.Run("worker_pool_creation", func(t *testing.T) {
		pool := NewWorkerPool(scanner, 4)
		if pool.WorkerCount() != 4 {
			t.Errorf("expected 4 workers, got %d", pool.WorkerCount())
		}
	})

	t.Run("worker_pool_auto_workers", func(t *testing.T) {
		pool := NewWorkerPool(scanner, 0)
		expected := max(runtime.NumCPU()/2, 1)
		if pool.WorkerCount() != expected {
			t.Errorf("expected %d workers, got %d", expected, pool.WorkerCount())
		}
	})
}

func TestParallelScanning(t *testing.T) {
	projectDir := "../../examples/demo"

	t.Run("scan_parallel_produces_same_results", func(t *testing.T) {
		// Sequential scan
		scanner1, err := New(projectDir)
		if err != nil {
			t.Fatalf("failed to create scanner: %v", err)
		}

		project1, err := scanner1.Scan()
		if err != nil {
			t.Fatalf("sequential scan failed: %v", err)
		}

		// Parallel scan
		scanner2, err := New(projectDir, WithParallel(4))
		if err != nil {
			t.Fatalf("failed to create scanner: %v", err)
		}

		project2, err := scanner2.Scan()
		if err != nil {
			t.Fatalf("parallel scan failed: %v", err)
		}

		// Compare results
		if len(project1.Controllers) != len(project2.Controllers) {
			t.Errorf("controller count mismatch: %d vs %d",
				len(project1.Controllers), len(project2.Controllers))
		}

		if len(project1.Providers) != len(project2.Providers) {
			t.Errorf("provider count mismatch: %d vs %d",
				len(project1.Providers), len(project2.Providers))
		}

		if len(project1.Middleware) != len(project2.Middleware) {
			t.Errorf("middleware count mismatch: %d vs %d",
				len(project1.Middleware), len(project2.Middleware))
		}

		if len(project1.Configs) != len(project2.Configs) {
			t.Errorf("config count mismatch: %d vs %d",
				len(project1.Configs), len(project2.Configs))
		}
	})

	t.Run("parallel_with_cache", func(t *testing.T) {
		cacheDir := t.TempDir()

		scanner, err := New(projectDir, WithParallel(4), WithCache(cacheDir))
		if err != nil {
			t.Fatalf("failed to create scanner: %v", err)
		}

		// First scan - populate cache
		project1, err := scanner.Scan()
		if err != nil {
			t.Fatalf("first scan failed: %v", err)
		}

		// Second scan - use cache
		scanner2, err := New(projectDir, WithParallel(4), WithCache(cacheDir))
		if err != nil {
			t.Fatalf("failed to create second scanner: %v", err)
		}

		project2, err := scanner2.Scan()
		if err != nil {
			t.Fatalf("second scan failed: %v", err)
		}

		if len(project1.Controllers) != len(project2.Controllers) {
			t.Error("cached parallel scan produced different results")
		}
	})
}

func BenchmarkParallelScanning(b *testing.B) {
	projectDir := "../../examples/demo"

	b.Run("Sequential", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			scanner, _ := New(projectDir)
			_, _ = scanner.Scan()
		}
	})

	b.Run("Parallel_2Workers", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			scanner, _ := New(projectDir, WithParallel(2))
			_, _ = scanner.Scan()
		}
	})

	b.Run("Parallel_4Workers", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			scanner, _ := New(projectDir, WithParallel(4))
			_, _ = scanner.Scan()
		}
	})

	b.Run("Parallel_Auto", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			scanner, _ := New(projectDir, WithParallel(0))
			_, _ = scanner.Scan()
		}
	})

	b.Run("Parallel_WithCache", func(b *testing.B) {
		cacheDir := b.TempDir()

		// Warm up cache
		scanner, _ := New(projectDir, WithParallel(4), WithCache(cacheDir))
		scanner.Scan()

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			scanner, _ := New(projectDir, WithParallel(4), WithCache(cacheDir))
			_, _ = scanner.Scan()
		}
	})
}

func BenchmarkWorkerPoolOverhead(b *testing.B) {
	projectDir := "../../examples/demo"
	scanner, _ := New(projectDir)

	b.Run("CreatePool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pool := NewWorkerPool(scanner, 4)
			_ = pool
		}
	})

	b.Run("StartAndClose", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pool := NewWorkerPool(scanner, 4)
			pool.Start()
			pool.Close()
		}
	})
}
