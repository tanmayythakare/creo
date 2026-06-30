//go:build windows

package process

import (
	"os/exec"
	"testing"
	"time"
)

func TestIsTerminalProcess(t *testing.T) {
	t.Run("terminal shell process", func(t *testing.T) {
		cmd := exec.Command("cmd.exe")
		err := cmd.Start()
		if err != nil {
			t.Fatalf("failed to start cmd.exe: %v", err)
		}
		defer func() { _ = cmd.Process.Kill() }()

		isTerm := IsTerminalProcess(cmd.Process.Pid)
		if !isTerm {
			t.Errorf("expected cmd.exe (PID %d) to be recognized as a terminal process", cmd.Process.Pid)
		}
	})

	t.Run("non-terminal process", func(t *testing.T) {
		// ping.exe is not on the terminal shells allowlist
		cmd := exec.Command("ping.exe", "127.0.0.1", "-n", "10")
		err := cmd.Start()
		if err != nil {
			t.Fatalf("failed to start ping.exe: %v", err)
		}
		defer func() { _ = cmd.Process.Kill() }()

		isTerm := IsTerminalProcess(cmd.Process.Pid)
		if isTerm {
			t.Errorf("expected ping.exe (PID %d) NOT to be recognized as a terminal process", cmd.Process.Pid)
		}
	})

	t.Run("non-existent process", func(t *testing.T) {
		isTerm := IsTerminalProcess(999999)
		if isTerm {
			t.Error("expected non-existent process to not be recognized as terminal")
		}
	})
}

func TestHasChildProcesses(t *testing.T) {
	t.Run("process with child", func(t *testing.T) {
		// Start cmd.exe and have it run ping.exe
		cmd := exec.Command("cmd.exe", "/c", "ping 127.0.0.1 -n 5")
		err := cmd.Start()
		if err != nil {
			t.Fatalf("failed to start cmd.exe: %v", err)
		}
		defer func() { _ = cmd.Process.Kill() }()

		// Give it a tiny bit of time to spawn the child ping.exe
		time.Sleep(200 * time.Millisecond)

		hasChildren, err := HasChildProcesses(cmd.Process.Pid)
		if err != nil {
			t.Fatalf("HasChildProcesses returned unexpected error: %v", err)
		}
		if !hasChildren {
			t.Errorf("expected parent cmd.exe (PID %d) to have active child processes", cmd.Process.Pid)
		}
	})

	t.Run("process with no children", func(t *testing.T) {
		// Spawn ping.exe directly, which runs standalone with no children
		cmd := exec.Command("ping.exe", "127.0.0.1", "-n", "5")
		err := cmd.Start()
		if err != nil {
			t.Fatalf("failed to start ping.exe: %v", err)
		}
		defer func() { _ = cmd.Process.Kill() }()

		hasChildren, err := HasChildProcesses(cmd.Process.Pid)
		if err != nil {
			t.Fatalf("HasChildProcesses returned unexpected error: %v", err)
		}
		if hasChildren {
			t.Errorf("expected standalone process (PID %d) to have no child processes", cmd.Process.Pid)
		}
	})

	t.Run("dead process", func(t *testing.T) {
		hasChildren, err := HasChildProcesses(999999)
		if err != nil {
			// CreateToolhelp32Snapshot doesn't error out on non-existent parent PIDs,
			// it just returns no entries.
		}
		if hasChildren {
			t.Error("expected non-existent process to have no children")
		}
	})
}
