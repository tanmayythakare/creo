package process

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/tanmayythakare/creo/internal/ui"
)

func TestIsRunning(t *testing.T) {
	t.Run("running terminal process with child", func(t *testing.T) {
		cmd := exec.Command("cmd.exe", "/c", "ping 127.0.0.1 -n 5")
		err := cmd.Start()
		if err != nil {
			t.Fatalf("failed to start cmd.exe: %v", err)
		}
		defer func() { _ = cmd.Process.Kill() }()
		time.Sleep(200 * time.Millisecond)

		reg := Registry{"api": cmd.Process.Pid}
		pid, running := IsRunning(reg, "api")

		if !running || pid != cmd.Process.Pid {
			t.Errorf("expected running=true and pid=%d, got running=%t, pid=%d", cmd.Process.Pid, running, pid)
		}
	})

	t.Run("running terminal process with no child", func(t *testing.T) {
		cmd := exec.Command("cmd.exe")
		err := cmd.Start()
		if err != nil {
			t.Fatalf("failed to start cmd.exe: %v", err)
		}
		defer func() { _ = cmd.Process.Kill() }()

		reg := Registry{"api": cmd.Process.Pid}
		_, running := IsRunning(reg, "api")

		// cmd.exe running alone has no children, so IsRunning should report false
		if running {
			t.Error("expected running=false for terminal process with no children")
		}
	})

	t.Run("dead process", func(t *testing.T) {
		reg := Registry{"api": 999999}
		_, running := IsRunning(reg, "api")
		if running {
			t.Error("expected running=false for dead process")
		}
	})

	t.Run("not in registry", func(t *testing.T) {
		reg := Registry{}
		_, running := IsRunning(reg, "api")
		if running {
			t.Error("expected running=false for service not in registry")
		}
	})
}

func TestVerifyPID(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("valid running service (terminal with child)", func(t *testing.T) {
		cmd := exec.Command("cmd.exe", "/c", "ping 127.0.0.1 -n 5")
		err := cmd.Start()
		if err != nil {
			t.Fatalf("failed to start cmd.exe: %v", err)
		}
		defer func() { _ = cmd.Process.Kill() }()
		time.Sleep(200 * time.Millisecond)

		reg := Registry{"api": cmd.Process.Pid}
		pid, running, err := VerifyPID(reg, "api", tempDir)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !running || pid != cmd.Process.Pid {
			t.Errorf("expected running=true, pid=%d, got running=%t, pid=%d", cmd.Process.Pid, running, pid)
		}
	})

	t.Run("terminal with no children (cleaned)", func(t *testing.T) {
		cmd := exec.Command("cmd.exe")
		err := cmd.Start()
		if err != nil {
			t.Fatalf("failed to start cmd.exe: %v", err)
		}
		defer func() { _ = cmd.Process.Kill() }()

		reg := Registry{"api": cmd.Process.Pid}
		_ = SetPID(tempDir, "api", cmd.Process.Pid)

		pid, running, err := VerifyPID(reg, "api", tempDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if running || pid != 0 {
			t.Errorf("expected running=false, pid=0, got running=%t, pid=%d", running, pid)
		}

		// Registry should have been cleaned
		r, _ := LoadRegistry(tempDir)
		if _, ok := r["api"]; ok {
			t.Error("expected registry to be cleaned")
		}
	})

	t.Run("PID reuse warning - clear accepted", func(t *testing.T) {
		// Our own test process is running, but it's not a terminal shell,
		// which simulates a reused PID.
		myPID := os.Getpid()
		reg := Registry{"api": myPID}
		_ = SetPID(tempDir, "api", myPID)

		// Mock user input: Yes, clear it
		ui.SetReader(strings.NewReader("y\n"))

		pid, running, err := VerifyPID(reg, "api", tempDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if running || pid != 0 {
			t.Errorf("expected running=false, pid=0, got running=%t, pid=%d", running, pid)
		}

		// Registry should have been cleaned
		r, _ := LoadRegistry(tempDir)
		if _, ok := r["api"]; ok {
			t.Error("expected registry to be cleared")
		}
	})

	t.Run("PID reuse warning - clear aborted", func(t *testing.T) {
		myPID := os.Getpid()
		reg := Registry{"api": myPID}
		_ = SetPID(tempDir, "api", myPID)

		// Mock user input: No, don't clear it
		ui.SetReader(strings.NewReader("n\n"))

		_, _, err := VerifyPID(reg, "api", tempDir)
		if err == nil {
			t.Fatal("expected error from aborted VerifyPID, got nil")
		}
		if !strings.Contains(err.Error(), "aborted due to PID reuse warning") {
			t.Errorf("expected PID reuse error, got: %v", err)
		}
	})
}

func TestStopService(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("stop active service - confirm yes", func(t *testing.T) {
		cmd := exec.Command("cmd.exe", "/c", "ping 127.0.0.1 -n 5")
		err := cmd.Start()
		if err != nil {
			t.Fatalf("failed to start cmd.exe: %v", err)
		}
		defer func() { _ = cmd.Process.Kill() }()
		time.Sleep(200 * time.Millisecond)

		reg := Registry{"api": cmd.Process.Pid}
		_ = SetPID(tempDir, "api", cmd.Process.Pid)

		// Mock confirmation to stop
		ui.SetReader(strings.NewReader("y\n"))

		wasKilled, pid, err := StopService(reg, "api", tempDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !wasKilled || pid != cmd.Process.Pid {
			t.Errorf("expected wasKilled=true, pid=%d, got wasKilled=%t, pid=%d", cmd.Process.Pid, wasKilled, pid)
		}

		// Ensure the process is dead (Wait checks exit status)
		// taskkill /F /T should have terminated it
		errChan := make(chan error, 1)
		go func() {
			errChan <- cmd.Wait()
		}()
		select {
		case <-errChan:
			// Process successfully killed/exited
		case <-time.After(1 * time.Second):
			t.Error("timeout waiting for process to terminate")
		}
	})

	t.Run("stop active service - confirm no", func(t *testing.T) {
		cmd := exec.Command("cmd.exe", "/c", "ping 127.0.0.1 -n 5")
		err := cmd.Start()
		if err != nil {
			t.Fatalf("failed to start cmd.exe: %v", err)
		}
		defer func() { _ = cmd.Process.Kill() }()
		time.Sleep(200 * time.Millisecond)

		reg := Registry{"api": cmd.Process.Pid}
		_ = SetPID(tempDir, "api", cmd.Process.Pid)

		// Mock confirmation: No
		ui.SetReader(strings.NewReader("n\n"))

		wasKilled, _, err := StopService(reg, "api", tempDir)
		if err == nil {
			t.Fatal("expected error from aborted stop, got nil")
		}
		if wasKilled {
			t.Error("expected wasKilled=false when aborted")
		}
	})

	t.Run("stop already stopped service", func(t *testing.T) {
		reg := Registry{"api": 999999}
		_ = SetPID(tempDir, "api", 999999)

		wasKilled, pid, err := StopService(reg, "api", tempDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if wasKilled || pid != 0 {
			t.Errorf("expected wasKilled=false, pid=0, got wasKilled=%t, pid=%d", wasKilled, pid)
		}
	})
}

func TestStopAll(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("stop all services", func(t *testing.T) {
		cmd1 := exec.Command("cmd.exe", "/c", "ping 127.0.0.1 -n 5")
		_ = cmd1.Start()
		defer func() { _ = cmd1.Process.Kill() }()

		cmd2 := exec.Command("cmd.exe", "/c", "ping 127.0.0.1 -n 5")
		_ = cmd2.Start()
		defer func() { _ = cmd2.Process.Kill() }()

		time.Sleep(200 * time.Millisecond)

		reg := Registry{
			"web": cmd1.Process.Pid,
			"api": cmd2.Process.Pid,
		}

		// Mock confirmation prompts for both services
		ui.SetReader(strings.NewReader("y\ny\n"))

		results := StopAll(reg, tempDir)
		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}

		if results["web"] != cmd1.Process.Pid {
			t.Errorf("expected web to be killed with PID %d, got %d", cmd1.Process.Pid, results["web"])
		}
		if results["api"] != cmd2.Process.Pid {
			t.Errorf("expected api to be killed with PID %d, got %d", cmd2.Process.Pid, results["api"])
		}
	})
}
