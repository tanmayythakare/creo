package process

import (
	"os"
	"strings"
	"testing"

	"github.com/tanmayythakare/creo/internal/ui"
)

func TestRegistry(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("save and read PID", func(t *testing.T) {
		err := SetPID(tempDir, "test-svc", 12345)
		if err != nil {
			t.Fatalf("unexpected error setting PID: %v", err)
		}

		reg, err := LoadRegistry(tempDir)
		if err != nil {
			t.Fatalf("unexpected error loading registry: %v", err)
		}
		if reg["test-svc"] != 12345 {
			t.Errorf("expected 12345, got %d", reg["test-svc"])
		}
	})

	t.Run("remove PID", func(t *testing.T) {
		err := RemovePID(tempDir, "test-svc")
		if err != nil {
			t.Fatalf("unexpected error removing PID: %v", err)
		}

		reg, err := LoadRegistry(tempDir)
		if err != nil {
			t.Fatalf("unexpected error loading registry: %v", err)
		}
		if _, ok := reg["test-svc"]; ok {
			t.Error("expected test-svc to be removed")
		}
	})

	t.Run("load non-existent registry", func(t *testing.T) {
		reg, err := LoadRegistry(t.TempDir()) // fresh dir
		if err != nil {
			t.Errorf("expected no error loading non-existent registry, got: %v", err)
		}
		if reg != nil && len(reg) > 0 {
			t.Errorf("expected empty registry")
		}
	})

	t.Run("IsPIDAlive", func(t *testing.T) {
		if !IsPIDAlive(os.Getpid()) {
			t.Error("expected current PID to be reported as alive")
		}
		if IsPIDAlive(999999) {
			t.Error("expected non-existent PID to be reported as dead")
		}
		if IsPIDAlive(-1) {
			t.Error("expected negative PID to be reported as dead")
		}
	})

	t.Run("LockRegistry and UnlockRegistry", func(t *testing.T) {
		lockDir := t.TempDir()

		// 1. Success Lock
		err := LockRegistry(lockDir)
		if err != nil {
			t.Fatalf("failed to lock registry: %v", err)
		}

		// 2. Lock again (already locked by us, an alive PID) -> should warning prompt
		// Mock input: No
		ui.SetReader(strings.NewReader("n\n"))
		err = LockRegistry(lockDir)
		if err == nil {
			t.Fatal("expected error locking already locked registry, got nil")
		}

		// Mock input: Yes (Force continue)
		ui.SetReader(strings.NewReader("y\n"))
		err = LockRegistry(lockDir)
		if err != nil {
			t.Fatalf("expected lock to succeed with force input, got: %v", err)
		}

		// 3. Unlock
		UnlockRegistry(lockDir)

		// 4. Lock again (should succeed immediately since lock is deleted)
		err = LockRegistry(lockDir)
		if err != nil {
			t.Fatalf("failed to lock after unlock: %v", err)
		}
		UnlockRegistry(lockDir)
	})
}

