package process

import (
	"os"
	"testing"

	"github.com/tanmayythakare/creo/internal/config"
)

func TestCleanStale(t *testing.T) {
	tempDir := t.TempDir()

	// Write registry with an alive PID (our own process) and a dead PID (999999)
	myPID := os.Getpid()
	deadPID := 999999

	err := SetPID(tempDir, "alive-service", myPID)
	if err != nil {
		t.Fatalf("failed to set alive service PID: %v", err)
	}

	err = SetPID(tempDir, "dead-service", deadPID)
	if err != nil {
		t.Fatalf("failed to set dead service PID: %v", err)
	}

	CleanStale(tempDir)

	reg, err := LoadRegistry(tempDir)
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}

	// The alive service should still be there
	if pid, ok := reg["alive-service"]; !ok || pid != myPID {
		t.Errorf("expected alive-service to remain with PID %d, got ok=%t, pid=%d", myPID, ok, pid)
	}

	// The dead service should have been pruned
	if _, ok := reg["dead-service"]; ok {
		t.Error("expected dead-service to be pruned by CleanStale")
	}
}

func TestCleanOrphans(t *testing.T) {
	tempDir := t.TempDir()

	err := SetPID(tempDir, "web", 1234)
	if err != nil {
		t.Fatalf("failed to set PID: %v", err)
	}
	err = SetPID(tempDir, "db", 5678)
	if err != nil {
		t.Fatalf("failed to set PID: %v", err)
	}

	// Config only lists "web"
	cfg := &config.Config{
		Services: map[string]config.Service{
			"web": {Command: "npm run dev"},
		},
	}

	CleanOrphans(tempDir, cfg)

	reg, err := LoadRegistry(tempDir)
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}

	// web service is not an orphan
	if _, ok := reg["web"]; !ok {
		t.Error("expected 'web' to remain in registry")
	}

	// db service was not in config, so it is an orphan and should be pruned
	if _, ok := reg["db"]; ok {
		t.Error("expected orphaned 'db' service to be pruned")
	}

	// Nil config safety check
	CleanOrphans(tempDir, nil)
}
