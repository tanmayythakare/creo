package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/tanmayythakare/creo/internal/bootstrap"
	"github.com/tanmayythakare/creo/internal/config"
	"github.com/tanmayythakare/creo/internal/process"
	"github.com/tanmayythakare/creo/internal/ui"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize creo in this repo",
	Long:  "Scans the repo, detects services, and walks you through creating creo.yml",
	RunE:  runInit,
}

var forceInit bool

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVar(&forceInit, "force", false, "overwrite existing creo.yml without asking")
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("✗ cannot get working directory: %w. Fix: Check directory permissions", err)
	}

	configPath := filepath.Join(cwd, "creo.yml")

	// Guard: already initialized
	if _, err := os.Stat(configPath); err == nil && !forceInit {
		overwrite := ui.Confirm("creo.yml already exists. Overwrite?", false)
		if !overwrite {
			ui.Info("Aborted. Existing creo.yml kept.")
			return nil
		}
	}

	// Lock registry during init to prevent race conditions
	if err := process.LockRegistry(cwd); err != nil {
		ui.Error(fmt.Sprintf("%v", err))
		return nil
	}
	defer process.UnlockRegistry(cwd)

	// Catch Ctrl+C / SIGINT cleanly
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	cancelChan := make(chan struct{})
	go func() {
		select {
		case <-sigChan:
			process.UnlockRegistry(cwd)
			fmt.Println()
			ui.Warn("Initialization aborted. No creo.yml was written.")
			os.Exit(0)
		case <-cancelChan:
			return
		}
	}()
	defer func() {
		close(cancelChan)
		signal.Stop(sigChan)
	}()

	ui.PrintBannerWithGlitch()

	// Phase 1: Scan
	var scan *bootstrap.ScanResult
	var scanErr error
	ui.SpinUntil("scanning repo", func() {
		scan, scanErr = bootstrap.Scan(cwd)
	})
	if scanErr != nil {
		return fmt.Errorf("✗ scan failed: %w. Fix: Check repository permissions", scanErr)
	}

	// Phase 2: Detect
	detected := bootstrap.Detect(scan)

	fmt.Println()
	fmt.Printf("  found %d services:\n\n", len(detected))

	maxNameLen := 12
	for _, d := range detected {
		if len(d.Name) > maxNameLen {
			maxNameLen = len(d.Name)
		}
	}

	nameCol := lipgloss.NewStyle().Width(maxNameLen)
	frameworkCol := lipgloss.NewStyle().Width(16)
	cmdCol := lipgloss.NewStyle().Width(30)

	for _, d := range detected {
		confSuffix := ""
		if d.Confidence == "low" {
			confSuffix = ui.Dim.Render(" (low confidence)")
		}
		fmt.Printf("  %s %s %s%s\n",
			nameCol.Render(ui.Truncate(d.Name, maxNameLen)),
			frameworkCol.Render(ui.Truncate(d.Framework, 16)),
			cmdCol.Render(ui.Truncate(d.SuggestedCmd, 30)),
			confSuffix,
		)
	}
	fmt.Println()

	printUndetectedHints(cwd, detected)

	var cfg *config.Config
	if len(detected) > 0 {
		looksRight := ui.Confirm("looks right?", true)
		fmt.Println()
		if looksRight {
			var wfName string
			for {
				wfName = ui.Ask("workflow name", "")
				wfName = strings.TrimSpace(wfName)
				if wfName == "" {
					ui.Warn("Workflow name cannot be empty or only spaces. Please try again.")
					continue
				}
				if strings.Contains(wfName, " ") {
					wfName = strings.ReplaceAll(wfName, " ", "-")
					ui.Info(fmt.Sprintf("Spaces in workflow name converted to hyphens: '%s'", wfName))
				}
				if bootstrap.IsBuiltinCommand(wfName) {
					ui.Warn(fmt.Sprintf("Workflow name '%s' conflicts with built-in command. Please try again.", wfName))
					continue
				}
				break
			}
			fmt.Println()

			projectName := filepath.Base(cwd)
			if projectName == `\` || projectName == "." || projectName == "/" || projectName == "" {
				projectName = ui.Ask("Project name", "my-project")
			}
			projectName = strings.TrimSpace(projectName)
			if projectName == "" {
				projectName = "my-project"
			}

			cfg = &config.Config{
				Project:   config.Project{Name: projectName},
				Services:  make(map[string]config.Service),
				Workflows: make(map[string][]string),
			}

			var serviceNames []string
			for _, d := range detected {
				cfg.Services[d.Name] = config.Service{
					Path:    d.Path,
					Command: d.SuggestedCmd,
				}
				serviceNames = append(serviceNames, d.Name)
			}
			cfg.Workflows[wfName] = serviceNames
		}
	}

	if cfg == nil {
		// Proceed to wizard
		var wizardErr error
		cfg, wizardErr = bootstrap.RunWizard(detected)
		if wizardErr != nil {
			return fmt.Errorf("✗ wizard error: %w. Fix: Try running creo init again", wizardErr)
		}
	}

	// Validate config isn't empty (0 services)
	if len(cfg.Services) == 0 {
		return fmt.Errorf("✗ cannot create config with 0 services. Fix: Add at least one service during setup")
	}

	// Validate none of the services have empty commands
	for name, svc := range cfg.Services {
		if strings.TrimSpace(svc.Command) == "" {
			return fmt.Errorf("✗ service '%s' cannot have an empty command. Fix: Provide a valid start command", name)
		}
	}

	// Validate none of the workflows have 0 services
	for wName, svcs := range cfg.Workflows {
		if len(svcs) == 0 {
			return fmt.Errorf("✗ workflow '%s' has 0 services. Fix: Add at least one service to the workflow", wName)
		}
	}

	// Stop catching signals since we are writing the file now
	signal.Stop(sigChan)

	// Phase 4: Write config
	if err := config.Write(cfg, configPath); err != nil {
		return fmt.Errorf("✗ failed to write creo.yml: %w. Fix: Check file permissions", err)
	}

	// Find the workflow name
	var wfName string
	for name := range cfg.Workflows {
		wfName = name
		break // just get the first one for the command hint
	}
	if wfName == "" {
		wfName = "boom"
	}

	ui.Success(fmt.Sprintf("creo.yml created — run: creo %s", wfName))
	fmt.Println()

	return nil
}

func printConfigSummary(cfg *config.Config) {
	ui.Section("Summary")
	ui.Info(fmt.Sprintf("Project: %s", cfg.Project.Name))
	ui.Info(fmt.Sprintf("Services: %d", len(cfg.Services)))
	for name, svc := range cfg.Services {
		ui.Info(fmt.Sprintf("  • %s → %s", name, svc.Command))
	}
	ui.Info(fmt.Sprintf("Workflows: %d", len(cfg.Workflows)))
	for name, svcs := range cfg.Workflows {
		ui.Info(fmt.Sprintf("  • %s → %v", name, svcs))
	}
	fmt.Println()
}

// printUndetectedHints checks for root-level directories that match common service
// folder names but weren't detected as services. Prints a soft hint — not a blocker.
// Intentionally excludes "app" and "ui" to avoid false positives with Next.js app/
// router directories and ui/ component libraries that aren't standalone services.
func printUndetectedHints(cwd string, detected []bootstrap.DetectedService) {
	commonNames := map[string]bool{
		"frontend": true, "client": true, "web": true,
		"backend": true, "server": true, "api": true,
	}

	entries, err := os.ReadDir(cwd)
	if err != nil {
		return
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := strings.ToLower(e.Name())
		if !commonNames[name] {
			continue
		}

		// Check if any detected service path starts with this folder
		covered := false
		for _, d := range detected {
			normalizedPath := filepath.ToSlash(d.Path)
			if normalizedPath == name || strings.HasPrefix(normalizedPath, name+"/") {
				covered = true
				break
			}
		}

		if !covered {
			ui.Info(fmt.Sprintf(
				"Noticed a '%s' folder but couldn't confirm its stack — you'll be able to add it manually in the next step.",
				e.Name(),
			))
		}
	}
}

