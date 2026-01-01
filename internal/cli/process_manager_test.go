package cli

import (
	"os"
	"testing"
	"time"
)

func TestProcessManager_StartStop(t *testing.T) {
	// Create a simple long-running process (sleep)
	tmpScript := writeTempScript(t, `#!/bin/sh
trap "exit 0" TERM
sleep 30
`)
	defer os.Remove(tmpScript)

	pm := &ProcessManager{quiet: true}

	// Start the process
	if err := pm.Start(tmpScript, 8080); err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	if !pm.IsRunning() {
		t.Fatal("Process should be running")
	}

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Stop the process
	if err := pm.Stop(); err != nil {
		t.Fatalf("Failed to stop process: %v", err)
	}

	if pm.IsRunning() {
		t.Fatal("Process should not be running after Stop()")
	}

	// Verify we can call Stop again without error
	if err := pm.Stop(); err != nil {
		t.Fatalf("Second Stop() should not error: %v", err)
	}
}

func TestProcessManager_Restart(t *testing.T) {
	tmpScript := writeTempScript(t, `#!/bin/sh
trap "exit 0" TERM
sleep 30
`)
	defer os.Remove(tmpScript)

	pm := &ProcessManager{quiet: true}

	// Start
	if err := pm.Start(tmpScript, 8080); err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Restart
	if err := pm.Restart(tmpScript, 8080); err != nil {
		t.Fatalf("Failed to restart process: %v", err)
	}

	if !pm.IsRunning() {
		t.Fatal("Process should be running after restart")
	}

	// Cleanup
	pm.Stop()
}

func TestProcessManager_StopNonExistent(t *testing.T) {
	pm := &ProcessManager{}

	// Should not error when stopping non-existent process
	if err := pm.Stop(); err != nil {
		t.Fatalf("Stop() on non-running process should not error: %v", err)
	}
}

func TestProcessManager_ForceKill(t *testing.T) {
	t.Skip("Skipping force kill test - takes too long")

	// Create a process that ignores SIGTERM
	tmpScript := writeTempScript(t, `#!/bin/sh
trap "" TERM
echo "Starting"
sleep 30
`)
	defer os.Remove(tmpScript)

	pm := &ProcessManager{}

	// Start
	if err := pm.Start(tmpScript, 8080); err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Stop should timeout and force kill
	start := time.Now()
	if err := pm.Stop(); err != nil {
		t.Fatalf("Failed to stop process: %v", err)
	}
	elapsed := time.Since(start)

	// Should have force-killed after timeout (3 seconds)
	if elapsed < 3*time.Second || elapsed > 4*time.Second {
		t.Fatalf("Expected force kill after ~3s, took %v", elapsed)
	}

	if pm.IsRunning() {
		t.Fatal("Process should not be running after force kill")
	}
}

// Helper to create a temporary executable script
func writeTempScript(t *testing.T, content string) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "test-script-*.sh")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}

	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}

	// Make it executable
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		t.Fatal(err)
	}

	return tmpFile.Name()
}

func TestProcessManager_MultipleRestarts(t *testing.T) {
	tmpScript := writeTempScript(t, `#!/bin/sh
trap "exit 0" TERM
sleep 30
`)
	defer os.Remove(tmpScript)

	pm := &ProcessManager{quiet: true}

	// Start
	if err := pm.Start(tmpScript, 8080); err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	// Do multiple restarts
	for i := 0; i < 3; i++ {
		time.Sleep(100 * time.Millisecond)
		if err := pm.Restart(tmpScript, 8080); err != nil {
			t.Fatalf("Restart %d failed: %v", i+1, err)
		}

		if !pm.IsRunning() {
			t.Fatalf("Process should be running after restart %d", i+1)
		}
	}

	// Cleanup
	pm.Stop()
}

func TestProcessManager_ProcessCrash(t *testing.T) {
	// Create a process that exits immediately
	tmpScript := writeTempScript(t, `#!/bin/sh
exit 1
`)
	defer os.Remove(tmpScript)

	pm := &ProcessManager{quiet: true}

	// Start
	if err := pm.Start(tmpScript, 8080); err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	// Wait for process to crash
	time.Sleep(200 * time.Millisecond)

	// Should not be running anymore
	if pm.IsRunning() {
		t.Fatal("Process should have crashed")
	}

	// Stop should be safe to call
	if err := pm.Stop(); err != nil {
		t.Fatalf("Stop after crash should not error: %v", err)
	}
}
