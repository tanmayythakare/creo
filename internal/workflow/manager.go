package workflow

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/tanmayythakare/creo/internal/config"
	"github.com/tanmayythakare/creo/internal/doctor"
	"github.com/tanmayythakare/creo/internal/process"
	"github.com/tanmayythakare/creo/internal/ui"
)

// Start runs a named workflow — cleans stale PIDs, checks ports, launches tabs
func Start(cfg *config.Config, workflowName string, serviceNames []string, root string, dryRun bool, verbose bool) error {
	// Clean up dead PIDs before starting
	process.CleanStale(root)

	// Print "creo — <workflowname>" header
	fmt.Println(ui.SectionHeader(workflowName))
	fmt.Println()

	// P0 check: wt.exe missing → clear error + run: creo doctor
	if _, err := exec.LookPath("wt.exe"); err != nil {
		ui.Error("wt.exe not found. Run: creo doctor")
		return nil
	}

	// Check WT version
	if _, warnMsg := doctor.CheckWTVersion(); warnMsg != "" {
		fmt.Println()
		ui.Warn("⚠ Windows Terminal version is outdated.\n  creo requires Windows Terminal 1.12 or higher for tab support.\n  Download latest: winget install Microsoft.WindowsTerminal")
		choice := ui.Ask("Continue anyway? [y/N]", "n")
		choice = strings.ToLower(strings.TrimSpace(choice))
		if choice != "y" && choice != "yes" {
			fmt.Println("  Aborting. Exiting cleanly.")
			return nil
		}
	}

	// Validate all services exist
	var valid []string
	for _, name := range serviceNames {
		if _, ok := cfg.Services[name]; !ok {
			ui.Warn(fmt.Sprintf("Service '%s' not found in creo.yml — skipping", name))
			continue
		}
		valid = append(valid, name)
	}

	if len(valid) == 0 {
		ui.Error("No valid services to start.")
		return nil
	}

	if dryRun {
		for _, name := range valid {
			ui.Success(fmt.Sprintf("dry-run: would launch %s", name))
		}
		ui.Success("Dry run complete: no terminals were launched.")
		return nil
	}

	reg, _ := process.LoadRegistry(root)
	start := time.Now()

	var toLaunch []string
	for _, name := range valid {
		svc := cfg.Services[name]

		// P0 check: service path doesn't exist or is not a directory → warn per service, skip it
		svcPath := svc.Path
		if svcPath == "" {
			svcPath = "."
		}
		absPath := filepath.Join(root, svcPath)
		fi, err := os.Stat(absPath)
		if os.IsNotExist(err) {
			ui.Warn(fmt.Sprintf("Service '%s' path '%s' does not exist. Skipping.", name, absPath))
			continue
		}
		if err == nil && !fi.IsDir() {
			ui.Warn(fmt.Sprintf("Service '%s' path '%s' is a file, not a directory. Skipping.", name, absPath))
			continue
		}

		svc = ResolveCommand(name, svc, absPath, root)
		cfg.Services[name] = svc

		// P0 check: services already running → warn + ask to skip or restart
		pid, running, errVerify := process.VerifyPID(reg, name, root)
		if errVerify != nil {
			ui.Error(errVerify.Error())
			return nil
		}
		if running {
			ui.Warn(fmt.Sprintf("Service '%s' is already running (PID %d).", name, pid))
			choice := ui.Ask("Skip launching or restart it? [S/r]", "s")
			choice = strings.ToLower(strings.TrimSpace(choice))
			if choice == "r" || choice == "restart" {
				ui.Info(fmt.Sprintf("Stopping %s (PID %d) before restart...", name, pid))
				killed, _, err := process.StopService(reg, name, root)
				if err != nil {
					ui.Error(fmt.Sprintf("Failed to stop %s: %v", name, err))
				} else if killed {
					ui.Success(fmt.Sprintf("Stopped %s", name))
				}
			} else {
				ui.Info(fmt.Sprintf("Skipping %s.", name))
				continue
			}
		}

		// P2 check: port conflict → warn which port, which service
		ports := findPorts(svc.Command)
		for _, p := range ports {
			if isPortInUse(p) {
				ui.Warn(fmt.Sprintf("Port %d is already in use. Service '%s' might fail to start.", p, name))
			}
		}

		toLaunch = append(toLaunch, name)
	}

	if len(toLaunch) > 0 {
		multi := pterm.DefaultMultiPrinter

		spinners := make(map[string]*pterm.SpinnerPrinter)

		dots2Spinner := pterm.DefaultSpinner
		dots2Spinner.Sequence = []string{"  ⣾", "  ⣽", "  ⣻", "  ⢿", "  ⡿", "  ⣟", "  ⣯", "  ⣷"}
		dots2Spinner.Delay = 80 * time.Millisecond
		dots2Spinner.Style = pterm.FgLightMagenta.ToStyle()
		dots2Spinner.ShowTimer = false
		dots2Spinner.SuccessPrinter = &pterm.PrefixPrinter{
			MessageStyle: pterm.NewStyle(),
			Prefix: pterm.Prefix{
				Style: pterm.NewStyle(pterm.FgGreen),
				Text:  "  ✓",
			},
		}
		dots2Spinner.FailPrinter = &pterm.PrefixPrinter{
			MessageStyle: pterm.NewStyle(),
			Prefix: pterm.Prefix{
				Style: pterm.NewStyle(pterm.FgRed),
				Text:  "  ✗",
			},
		}

		for _, name := range toLaunch {
			sp := dots2Spinner
			sp.Writer = multi.NewWriter()
			spActive, _ := sp.Start(name)
			spinners[name] = spActive
		}

		if _, err := multi.Start(); err != nil {
			return fmt.Errorf("failed to start multi-printer: %w", err)
		}

		// Launch all services in a single wt.exe invocation to avoid concurrent race conditions.
		// This builds one command with semicolons separating tabs, so wt.exe creates all tabs atomically.
		launchErr := LaunchAll(cfg, toLaunch, root, verbose)

		failed := make(map[string]bool)
		if launchErr != nil {
			// If the single wt.exe invocation failed, mark all services as failed
			for _, name := range toLaunch {
				failed[name] = true
				sp := spinners[name]
				sp.Fail(fmt.Sprintf("%s: %v", name, launchErr))
			}
		}

		// Poll for PIDs (LaunchAll's CollectAndRegisterPIDs already ran, but we need
		// to update spinners for any that completed there and catch any remaining)
		endTime := time.Now().Add(30 * time.Second)
		pending := make(map[string]bool)
		for _, name := range toLaunch {
			if !failed[name] {
				// Check if PID was already collected by LaunchAll's CollectAndRegisterPIDs
				latestReg, _ := process.LoadRegistry(root)
				if pid, ok := latestReg[name]; ok && pid > 0 {
					sp := spinners[name]
					sp.Success(name)
				} else {
					pending[name] = true
				}
			}
		}

		for time.Now().Before(endTime) && len(pending) > 0 {
			time.Sleep(100 * time.Millisecond)

			for name := range pending {
				pidFile := filepath.Join(root, ".creo", fmt.Sprintf("tmp_%s.pid", name))
				content, err := os.ReadFile(pidFile)
				if err == nil {
					pidStr := strings.TrimSpace(string(content))
					if pid, err := strconv.Atoi(pidStr); err == nil && pid > 0 {
						_ = process.SetPID(root, name, pid)
						_ = os.Remove(pidFile)
						delete(pending, name)
						sp := spinners[name]
						sp.Success(name)
					}
				}
			}
		}

		// Mark any remaining services that timed out as failed
		for name := range pending {
			pidFile := filepath.Join(root, ".creo", fmt.Sprintf("tmp_%s.pid", name))
			_ = os.Remove(pidFile)
			sp := spinners[name]
			sp.Fail(fmt.Sprintf("%s: timed out waiting for process to start", name))
		}

		_, _ = multi.Stop()

		// Wait a small buffer time (e.g., 500ms) to detect immediate startup crashes
		time.Sleep(500 * time.Millisecond)

		// Print active services & clickable URLs summary
		var startedServices []string
		var crashedServices []string
		reg, _ := process.LoadRegistry(root)

		for _, name := range toLaunch {
			if !failed[name] && !pending[name] {
				_, running := process.IsRunning(reg, name)
				if running {
					startedServices = append(startedServices, name)
				} else {
					crashedServices = append(crashedServices, name)
				}
			}
		}

		if len(startedServices) > 0 {
			fmt.Println()
			ui.Section("Active Services & Logs")
			for _, name := range startedServices {
				svc := cfg.Services[name]
				ports := findPorts(svc.Command)
				if len(ports) > 0 {
					var urls []string
					for _, p := range ports {
						urls = append(urls, fmt.Sprintf("http://localhost:%d", p))
					}
					fmt.Printf("  • %s: %s (Logs in Windows Terminal tab)\n", ui.Bold.Render(name), strings.Join(urls, ", "))
				} else {
					fmt.Printf("  • %s: Started (Logs in Windows Terminal tab)\n", ui.Bold.Render(name))
				}
			}
		}

		if len(crashedServices) > 0 {
			fmt.Println()
			ui.Section("Failed to Start (Crashed)")
			for _, name := range crashedServices {
				fmt.Printf("  • %s: %s\n", ui.Bold.Render(name), ui.Dim.Render("Process exited. Check Windows Terminal tab for error logs."))
			}
		}
	}

	duration := time.Since(start).Seconds()
	fmt.Println()
	fmt.Printf("  Done in %.1fs\n", duration)
	fmt.Println()

	return nil
}

// Extract ports from command (looks for 4-digit or 5-digit numbers)
func findPorts(command string) []int {
	re := regexp.MustCompile(`\b([1-9]\d{3,4})\b`)
	matches := re.FindAllString(command, -1)
	var ports []int
	seen := make(map[int]bool)
	for _, m := range matches {
		if p, err := strconv.Atoi(m); err == nil {
			if !seen[p] {
				seen[p] = true
				ports = append(ports, p)
			}
		}
	}
	return ports
}

// Check if port is in use
func isPortInUse(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 200*time.Millisecond)
	if err == nil {
		conn.Close()
		return true
	}
	return false
}

// ResolveCommand attempts to find local wrappers (mvnw, gradlew, npm bins) if the executable is missing from PATH
func ResolveCommand(name string, svc config.Service, absPath string, root string) config.Service {
	cmdParts := strings.Fields(svc.Command)
	if len(cmdParts) == 0 {
		return svc
	}
	cmdExe := cmdParts[0]
	for _, part := range cmdParts {
		if !strings.Contains(part, "=") {
			cmdExe = part
			break
		}
	}

	targetExe := strings.ToLower(cmdExe)
	if _, err := exec.LookPath(targetExe); err == nil {
		return svc // Exists in PATH
	}

	var wrapperPath string
	var wrapperFile string
	var toolName string

	// 1. Maven / Gradle wrappers
	if targetExe == "mvn" || targetExe == "gradle" {
		var candidateFiles []string
		if targetExe == "mvn" {
			candidateFiles = []string{"mvnw.cmd", "mvnw"}
			toolName = "Maven wrapper"
		} else {
			candidateFiles = []string{"gradlew.bat", "gradlew"}
			toolName = "Gradle wrapper"
		}

		for _, file := range candidateFiles {
			p := filepath.Join(absPath, file)
			if _, errStat := os.Stat(p); errStat == nil {
				wrapperPath = filepath.Join(".", file)
				wrapperFile = file
				break
			}
		}
		if wrapperPath == "" {
			for _, file := range candidateFiles {
				p := filepath.Join(root, file)
				if _, errStat := os.Stat(p); errStat == nil {
					wrapperPath = fmt.Sprintf(`"%s"`, p)
					wrapperFile = file
					break
				}
			}
		}
	}

	// 2. Node.js local binaries (e.g. vite, next, tsc inside node_modules/.bin)
	if wrapperPath == "" {
		nodeBinPath := filepath.Join("node_modules", ".bin")
		for _, ext := range []string{".cmd", ".bat", ".ps1", ""} {
			p := filepath.Join(absPath, nodeBinPath, cmdExe+ext)
			if _, errStat := os.Stat(p); errStat == nil {
				wrapperPath = filepath.Join(".", nodeBinPath, cmdExe+ext)
				wrapperFile = filepath.Base(p)
				toolName = "Node package binary"
				break
			}
			pRoot := filepath.Join(root, nodeBinPath, cmdExe+ext)
			if _, errStat := os.Stat(pRoot); errStat == nil {
				wrapperPath = fmt.Sprintf(`"%s"`, pRoot)
				wrapperFile = filepath.Base(pRoot)
				toolName = "Node package binary"
				break
			}
		}
	}

	// 3. PHP Composer local binaries (e.g. phpunit inside vendor/bin)
	if wrapperPath == "" {
		composerBinPath := filepath.Join("vendor", "bin")
		for _, ext := range []string{".bat", ".cmd", ""} {
			p := filepath.Join(absPath, composerBinPath, cmdExe+ext)
			if _, errStat := os.Stat(p); errStat == nil {
				wrapperPath = filepath.Join(".", composerBinPath, cmdExe+ext)
				wrapperFile = filepath.Base(p)
				toolName = "PHP Composer binary"
				break
			}
			pRoot := filepath.Join(root, composerBinPath, cmdExe+ext)
			if _, errStat := os.Stat(pRoot); errStat == nil {
				wrapperPath = fmt.Sprintf(`"%s"`, pRoot)
				wrapperFile = filepath.Base(pRoot)
				toolName = "PHP Composer binary"
				break
			}
		}
	}

	// 4. Python Virtual Environments (venv, .venv, env)
	if wrapperPath == "" && (strings.HasPrefix(targetExe, "python") || strings.HasPrefix(targetExe, "pip")) {
		for _, envDir := range []string{".venv", "venv", "env"} {
			p := filepath.Join(absPath, envDir, "Scripts", cmdExe+".exe")
			if _, errStat := os.Stat(p); errStat == nil {
				wrapperPath = filepath.Join(".", envDir, "Scripts", cmdExe+".exe")
				wrapperFile = filepath.Base(p)
				toolName = "Python virtualenv binary"
				break
			}
			pRoot := filepath.Join(root, envDir, "Scripts", cmdExe+".exe")
			if _, errStat := os.Stat(pRoot); errStat == nil {
				wrapperPath = fmt.Sprintf(`"%s"`, pRoot)
				wrapperFile = filepath.Base(pRoot)
				toolName = "Python virtualenv binary"
				break
			}
		}
	}

	// Apply replacement if local wrapper/binary resolved successfully
	if wrapperPath != "" {
		lowerCmd := strings.ToLower(svc.Command)
		idx := strings.Index(lowerCmd, targetExe)
		if idx != -1 {
			svc.Command = svc.Command[:idx] + wrapperPath + svc.Command[idx+len(targetExe):]
		} else {
			svc.Command = strings.Replace(svc.Command, cmdExe, wrapperPath, 1)
		}
		ui.Info(fmt.Sprintf("Info: '%s' not found in PATH. Using local %s '%s' for service '%s'.", targetExe, toolName, wrapperFile, name))
	} else {
		if !strings.HasPrefix(cmdExe, ".") && !strings.Contains(cmdExe, "/") && !strings.Contains(cmdExe, "\\") {
			existsLocally := false
			for _, ext := range []string{"", ".exe", ".bat", ".cmd", ".com"} {
				if _, errStat := os.Stat(filepath.Join(absPath, cmdExe+ext)); errStat == nil {
					existsLocally = true
					break
				}
			}
			if !existsLocally {
				ui.Warn(fmt.Sprintf("Executable '%s' not found in PATH. Service '%s' might fail to start.", cmdExe, name))
			}
		}
	}

	return svc
}
