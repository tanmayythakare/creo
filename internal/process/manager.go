package process

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/tanmayythakare/creo/internal/ui"
)

// IsRunning checks if a service's PID is alive and belongs to an active terminal shell running a command
func IsRunning(reg Registry, name string) (int, bool) {
	pid, ok := reg[name]
	if !ok || pid <= 0 {
		return 0, false
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return 0, false
	}

	// On Windows, FindProcess always succeeds — send signal 0 to check
	err = checkProcess(proc)
	if err != nil {
		return 0, false
	}

	isTerminal := IsTerminalProcess(pid)
	if isTerminal {
		hasChildren, err := HasChildProcesses(pid)
		if err != nil || !hasChildren {
			return 0, false
		}
	}

	return pid, true
}

// VerifyPID checks if the recorded PID is stale or reused, and prompts the user to clear it if reused.
// Returns: (pid, running, error)
func VerifyPID(reg Registry, name, root string) (int, bool, error) {
	pid, ok := reg[name]
	if !ok || pid <= 0 {
		return 0, false, nil
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		_ = RemovePID(root, name)
		return 0, false, nil
	}

	if checkProcess(proc) != nil {
		_ = RemovePID(root, name)
		return 0, false, nil
	}

	if IsTerminalProcess(pid) {
		hasChildren, err := HasChildProcesses(pid)
		if err == nil && hasChildren {
			return pid, true, nil
		}
		_ = RemovePID(root, name)
		return 0, false, nil
	}

	// PID reuse detected!
	fmt.Println()
	ui.Warn(fmt.Sprintf("⚠ PID %d is registered for '%s' but appears to belong to a different process.\n  This usually means the service crashed and the PID was reused by the OS.", pid, name))
	choice := ui.Ask("Clear it from registry? [Y/n]", "y")
	choice = strings.ToLower(strings.TrimSpace(choice))
	if choice == "y" || choice == "yes" || choice == "" {
		_ = RemovePID(root, name)
		fmt.Printf("  Registry cleared for '%s'.\n", name)
		return 0, false, nil
	}

	return pid, false, fmt.Errorf("✗ aborted due to PID reuse warning")
}

// StopService kills a single service by name
// Returns: (wasKilled, pid, error)
func StopService(reg Registry, name, root string) (bool, int, error) {
	pid, running, err := VerifyPID(reg, name, root)
	if err != nil {
		return false, pid, err
	}
	if !running {
		if pid > 0 {
			_ = RemovePID(root, name)
		}
		return false, pid, nil
	}

	fmt.Println()
	ui.Warn(fmt.Sprintf("⚠ Stopping '%s' will kill its entire process tree.\n  This cannot be undone.", name))
	choice := ui.Ask("Proceed? [Y/n]", "y")
	choice = strings.ToLower(strings.TrimSpace(choice))
	if choice != "y" && choice != "yes" && choice != "" {
		return false, pid, fmt.Errorf("✗ service stop aborted by user")
	}

	// Kill entire process tree
	cmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", pid))
	if err := cmd.Run(); err != nil {
		// Fallback to proc.Kill if taskkill fails
		proc, errProc := os.FindProcess(pid)
		if errProc == nil {
			_ = proc.Kill()
		}
	}

	// Wait/confirm process death (up to 1.5s) to prevent rapid restart race conditions
	for i := 0; i < 15; i++ {
		_, stillRunning := IsRunning(reg, name)
		if !stillRunning {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	_ = RemovePID(root, name)
	return true, pid, nil
}

// StopAll kills every tracked service in the registry and returns map of service -> running pid if killed, or 0 if already stopped
func StopAll(reg Registry, root string) map[string]int {
	results := make(map[string]int)
	for name := range reg {
		killed, pid, _ := StopService(reg, name, root)
		if killed {
			results[name] = pid
		} else {
			results[name] = 0
		}
	}
	return results
}

// StopServiceSilent kills a single service without interactive prompts.
// Used by commands like 'creo drop' where user confirmation was already provided.
func StopServiceSilent(reg Registry, name, root string) (bool, int) {
	pid, ok := reg[name]
	if !ok || pid <= 0 {
		return false, 0
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		_ = RemovePID(root, name)
		return false, 0
	}

	if checkProcess(proc) != nil {
		_ = RemovePID(root, name)
		return false, 0
	}

	// Kill entire process tree
	cmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", pid))
	if err := cmd.Run(); err != nil {
		proc, errProc := os.FindProcess(pid)
		if errProc == nil {
			_ = proc.Kill()
		}
	}

	// Wait/confirm process death (up to 1.5s)
	for i := 0; i < 15; i++ {
		_, stillRunning := IsRunning(reg, name)
		if !stillRunning {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	_ = RemovePID(root, name)
	return true, pid
}

// StopAllSilent kills every tracked service without interactive prompts.
// Returns a map of service name -> PID for services that were actually killed.
func StopAllSilent(reg Registry, root string) map[string]int {
	results := make(map[string]int)
	for name := range reg {
		killed, pid := StopServiceSilent(reg, name, root)
		if killed {
			results[name] = pid
		}
	}
	return results
}

