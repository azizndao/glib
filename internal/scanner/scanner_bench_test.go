package scanner

import (
	"runtime"
	"testing"
)

// BenchmarkScanMemory measures memory usage when scanning the demo project
func BenchmarkScanMemory(b *testing.B) {
	projectDir := "../../examples/demo"

	b.ReportAllocs()
	b.ResetTimer()

	var m runtime.MemStats
	var totalAlloc uint64

	for i := 0; i < b.N; i++ {
		runtime.GC()
		runtime.ReadMemStats(&m)
		before := m.Alloc

		scanner, err := New(projectDir)
		if err != nil {
			b.Fatal(err)
		}

		project, err := scanner.Scan()
		if err != nil {
			b.Fatal(err)
		}

		runtime.ReadMemStats(&m)
		after := m.Alloc

		totalAlloc += (after - before)

		// Keep project alive to prevent premature GC
		_ = project
	}

	// Report average memory usage in MB
	avgMB := float64(totalAlloc) / float64(b.N) / 1024 / 1024
	b.ReportMetric(avgMB, "MB/scan")
}

// BenchmarkScanSpeed measures scanning speed
func BenchmarkScanSpeed(b *testing.B) {
	projectDir := "../../examples/demo"

	scanner, err := New(projectDir)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := scanner.Scan()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkScanDetailedMemory provides detailed memory breakdown
func BenchmarkScanDetailedMemory(b *testing.B) {
	projectDir := "../../examples/demo"

	b.Run("FullScan", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			scanner, _ := New(projectDir)
			project, err := scanner.Scan()
			if err != nil {
				b.Fatal(err)
			}
			_ = project
		}
	})

	b.Run("ScannerCreation", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			scanner, _ := New(projectDir)
			_ = scanner
		}
	})
}
