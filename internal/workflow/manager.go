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
	"github.com/tanmaythakare/creo/internal/config"
	"github.com/tanmaythakare/creo/internal/doctor"
	"github.com/tanmaythakare/creo/internal/process"
	"github.com/tanmaythakare/creo/internal/ui"
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

		type launchResult struct {
			name string
			err  error
		}
		resultsChan := make(chan launchResult, len(toLaunch))

		for _, name := range toLaunch {
			go func(n string) {
				svc := cfg.Services[n]
				err := LaunchTab(n, svc, root, verbose)
				resultsChan <- launchResult{name: n, err: err}
			}(name)
		}

		for i := 0; i < len(toLaunch); i++ {
			res := <-resultsChan
			sp := spinners[res.name]
			if res.err != nil {
				sp.Fail(fmt.Sprintf("%s: %v", res.name, res.err))
			} else {
				sp.Success(res.name)
			}
		}

		_, _ = multi.Stop()
	}

	duration := time.Since(start).Seconds()
	fmt.Println()
	fmt.Printf("  done in %.1fs\n", duration)
	fmt.Println()

	return nil
}

// Extract ports from command (looks for 4-digit or 5-digit numbers)
func findPorts(command string) []int {
	re := regexp.MustCompile(`\b(3\d{3}|4\d{3}|5\d{3}|8\d{3}|9\d{3})\b`)
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
