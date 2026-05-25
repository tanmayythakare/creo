package workflow

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tanmaythakare/creo/internal/config"
	"github.com/tanmaythakare/creo/internal/process"
	"github.com/tanmaythakare/creo/internal/ui"
)

// LaunchTab opens a new Windows Terminal tab for a single service
func LaunchTab(name string, svc config.Service, root string, verbose bool) error {
	svcPath := svc.Path
	if svcPath == "" {
		svcPath = "."
	}
	absPath := filepath.Join(root, svcPath)

	if _, err := exec.LookPath("wt.exe"); err != nil {
		return fmt.Errorf("wt.exe not found. Run: creo doctor")
	}

	// Ensure .creo exists
	_ = os.MkdirAll(filepath.Join(root, ".creo"), 0755)

	pidFile := filepath.Join(root, ".creo", fmt.Sprintf("tmp_%s.pid", name))
	writePIDCmd := fmt.Sprintf(`powershell -NoProfile -Command (Get-CimInstance Win32_Process -Filter """ProcessId=$PID""").ParentProcessId ^| Out-File -FilePath '%s' -Encoding ascii`, pidFile)
	shellCmd := fmt.Sprintf(`%s && cd /d "%s" && %s`, writePIDCmd, absPath, svc.Command)

	args := []string{
		"new-tab",
		"--title", name,
		"cmd", "/k", shellCmd,
	}

	if verbose {
		cmdStr := "wt.exe " + strings.Join(args, " ")
		fmt.Printf("  %s\n", ui.Dim.Render("Verbose: Launching: "+cmdStr))
	}

	cmd := exec.Command("wt.exe", args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("✗ failed to start wt.exe: %v. Fix: Check if Windows Terminal is correctly installed", err)
	}

	CollectAndRegisterPIDs(root, []string{name})
	return nil
}

// buildWTArgs builds a single wt.exe invocation for multiple services
// Uses semicolons to chain tabs in one wt.exe call
func buildWTArgs(services []serviceEntry, root string) []string {
	args := []string{}

	for i, entry := range services {
		absPath := filepath.Join(root, entry.svc.Path)
		pidFile := filepath.Join(root, ".creo", fmt.Sprintf("tmp_%s.pid", entry.name))
		writePIDCmd := fmt.Sprintf(`powershell -NoProfile -Command (Get-CimInstance Win32_Process -Filter """ProcessId=$PID""").ParentProcessId ^| Out-File -FilePath '%s' -Encoding ascii`, pidFile)
		escapedCommand := strings.ReplaceAll(entry.svc.Command, `"`, `\"`)
		shellCmd := fmt.Sprintf(`%s && cd /d "%s" && %s`, writePIDCmd, absPath, escapedCommand)

		if i == 0 {
			// First tab: new-tab
			args = append(args, "new-tab", "--title", entry.name, "cmd", "/k", shellCmd)
		} else {
			// Additional tabs: semicolon separator then new-tab
			args = append(args, ";", "new-tab", "--title", entry.name, "cmd", "/k", shellCmd)
		}
	}

	return args
}

type serviceEntry struct {
	name string
	svc  config.Service
}

// LaunchAll opens all services in separate tabs in one wt.exe call
func LaunchAll(cfg *config.Config, serviceNames []string, root string, verbose bool) error {
	var entries []serviceEntry
	for _, name := range serviceNames {
		svc, ok := cfg.Services[name]
		if !ok {
			continue
		}
		entries = append(entries, serviceEntry{name: name, svc: svc})
	}

	if len(entries) == 0 {
		return fmt.Errorf("✗ no valid services to launch. Fix: Check creo.yml services configuration")
	}

	if _, err := exec.LookPath("wt.exe"); err != nil {
		return fmt.Errorf("✗ wt.exe not found. Fix: Run: creo doctor")
	}

	// Ensure .creo exists
	_ = os.MkdirAll(filepath.Join(root, ".creo"), 0755)

	args := buildWTArgs(entries, root)

	if verbose {
		cmdStr := "wt.exe " + strings.Join(args, " ")
		fmt.Printf("Verbose: Executing command: %s\n", cmdStr)
	}

	cmd := exec.Command("wt.exe", args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("✗ failed to start wt.exe: %v. Fix: Check if Windows Terminal is correctly installed", err)
	}

	CollectAndRegisterPIDs(root, serviceNames)
	return nil
}

// CollectAndRegisterPIDs polls for temporary PID files, reads them, saves them to registry, and deletes them
func CollectAndRegisterPIDs(root string, serviceNames []string) {
	endTime := time.Now().Add(15 * time.Second)

	pending := make(map[string]bool)
	for _, name := range serviceNames {
		pending[name] = true
	}

	start := time.Now()
	i := 0
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	for time.Now().Before(endTime) && len(pending) > 0 {
		time.Sleep(100 * time.Millisecond)
		i++

		// Print progress message on the same line
		elapsed := int(time.Since(start).Seconds())
		var pendingNames []string
		for name := range pending {
			pendingNames = append(pendingNames, name)
		}
		frame := frames[i%len(frames)]
		fmt.Printf("\r  %s waiting for %s pid... (%ds)   ", frame, strings.Join(pendingNames, ", "), elapsed)

		for name := range pending {
			pidFile := filepath.Join(root, ".creo", fmt.Sprintf("tmp_%s.pid", name))
			content, err := os.ReadFile(pidFile)
			if err == nil {
				pidStr := strings.TrimSpace(string(content))
				if pid, err := strconv.Atoi(pidStr); err == nil && pid > 0 {
					_ = process.SetPID(root, name, pid)
					_ = os.Remove(pidFile)
					delete(pending, name)
				}
			}
		}
	}

	// Clear the progress line
	fmt.Print("\r                                                                          \r")

	// Clean up any remaining temp files in case of timeout
	for name := range pending {
		pidFile := filepath.Join(root, ".creo", fmt.Sprintf("tmp_%s.pid", name))
		_ = os.Remove(pidFile)
	}
}
