package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/tanmayythakare/creo/internal/config"
	"github.com/tanmayythakare/creo/internal/doctor"
	"github.com/tanmayythakare/creo/internal/ui"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check that all required tools are installed",
	Long:  "Scans creo.yml and verifies every tool needed to run your services is available",
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

type checkResult struct {
	Name        string
	Available   bool
	Version     string
	InstallHint string
	Warning     string
}

func runDoctor(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	cfg, err := config.Load(cwd)
	if err != nil {
		if _, statErr := os.Getwd(); statErr == nil {
			if _, statErr := os.Stat(filepath.Join(cwd, "creo.yml")); statErr == nil {
				ui.Error(fmt.Sprintf("Invalid creo.yml configuration: %v", err))
				return nil
			}
		}
		ui.Warn("No creo.yml found. Running system checks only. Run: creo init")
		cfg = nil
	}

	report := doctor.Run(cfg)

	fmt.Println(ui.SectionHeader("doctor"))
	fmt.Println()

	// Environment warnings (WSL, Docker, OneDrive, etc.)
	if len(report.Warnings) > 0 {
		fmt.Printf("  SYSTEM WARNINGS %s\n", ui.Divider)
		for _, w := range report.Warnings {
			ui.Warn(w)
		}
		fmt.Println()
	}

	var passed []checkResult
	var failed []checkResult

	wtHint := "winget install Microsoft.WindowsTerminal"
	seenPassed := make(map[string]bool)
	seenFailed := make(map[string]bool)

	// WT Check
	if report.WTAvailable {
		passed = append(passed, checkResult{Name: "wt.exe", Available: true, Version: report.WTVersion, Warning: report.WTWarning})
		seenPassed["wt.exe"] = true
	} else {
		failed = append(failed, checkResult{Name: "wt.exe", Available: false, InstallHint: wtHint})
		seenFailed["wt.exe"] = true
	}

	// Service checks
	for _, sr := range report.Services {
		for _, c := range sr.Checks {
			if c.Available {
				if !seenPassed[c.Tool] {
					passed = append(passed, checkResult{Name: c.Tool, Available: true, Version: c.Version, Warning: c.Warning})
					seenPassed[c.Tool] = true
				}
			} else {
				if !seenFailed[c.Tool] {
					failed = append(failed, checkResult{Name: c.Tool, Available: false, InstallHint: c.InstallHint})
					seenFailed[c.Tool] = true
				}
			}
		}
	}

	toolCol := lipgloss.NewStyle().Width(10)

	if len(passed) > 0 {
		fmt.Printf("  PASSED %s\n", ui.Divider)
		for _, p := range passed {
			versionText := ""
			if p.Version != "" {
				versionText = p.Version
			}
			fmt.Printf("  %s %s%s\n", ui.SymCheck, toolCol.Render(p.Name), ui.Dim.Render(versionText))
			if p.Warning != "" {
				fmt.Printf("    %s %s\n", ui.Warning.Render("⚠"), p.Warning)
			}
		}
		fmt.Println()
	}

	if len(failed) > 0 {
		fmt.Printf("  FAILED %s\n", ui.Divider)
		for _, f := range failed {
			fmt.Printf("  %s %s%s\n", ui.SymCross, toolCol.Render(f.Name), ui.Dim.Render("not found"))
			if f.InstallHint != "" {
				fmt.Printf("    %s %s\n", ui.SymArrow, f.InstallHint)
			}
		}
		fmt.Println()
	}

	if len(failed) > 0 {
		issuesText := "issue found"
		if len(failed) > 1 {
			issuesText = "issues found"
		}
		fmt.Printf("  %s\n\n", ui.Failure.Render(fmt.Sprintf("%d %s", len(failed), issuesText)))
	} else {
		fmt.Printf("  %s\n\n", ui.SymCheck+" all checks passed")
	}

	return nil
}
