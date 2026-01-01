package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkStreamingVsRegular(b *testing.B) {
	// Create test project
	tmpDir := b.TempDir()

	goMod := `module testproject

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		b.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create multiple provider files
	for i := 0; i < 10; i++ {
		code := `package main

// @Provider singleton
func NewService` + string(rune('A'+i)) + `() *Service {
	return &Service{}
}

type Service struct{}
`
		filename := filepath.Join(tmpDir, "provider"+string(rune('a'+i))+".go")
		if err := os.WriteFile(filename, []byte(code), 0644); err != nil {
			b.Fatalf("Failed to create file: %v", err)
		}
	}

	b.Run("Regular", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			scanner, _ := New(tmpDir)
			_, err := scanner.Scan()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Streaming", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			scanner, _ := New(tmpDir)
			events := make(chan StreamEvent, 100)

			// Consume events
			go func() {
				for range events {
					// Process events as they come
				}
			}()

			_, err := scanner.ScanWithStream(events)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkTimeToFirstResult measures how quickly we get the first useful event
func BenchmarkTimeToFirstResult(b *testing.B) {
	tmpDir := b.TempDir()

	goMod := `module testproject

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		b.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create provider file
	providerCode := `package main

// @Provider singleton
func NewDatabase() *Database {
	return &Database{}
}

type Database struct{}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "provider.go"), []byte(providerCode), 0644); err != nil {
		b.Fatalf("Failed to create provider.go: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		scanner, _ := New(tmpDir)
		events := make(chan StreamEvent, 10)

		done := make(chan bool)
		go func() {
			// Stop timing when we get first provider
			firstProvider := false
			for event := range events {
				if event.Type == EventProvider && !firstProvider {
					firstProvider = true
					b.StopTimer()
				}
			}
			done <- true
		}()

		b.StartTimer()
		scanner.ScanWithStream(events)
		<-done
	}
}
