package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/tanmayythakare/creo/internal/process"
	"github.com/tanmayythakare/creo/internal/ui"
)

var dropCmd = &cobra.Command{
	Use:     "drop",
	Aliases: []string{"delete"},
	Short:   "Remove Creo configuration and state from this project",
	Long:    "Stops running services, removes creo.yml and .creo/, and cleans up all Creo state from the current directory.",
	RunE:    runDrop,
}

var forceDrop bool

func init() {
	rootCmd.AddCommand(dropCmd)
	dropCmd.Flags().BoolVarP(&forceDrop, "force", "f", false, "skip confirmation prompt")
}

func runDrop(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("✗ cannot get working directory: %w. Fix: Check directory permissions", err)
	}

	// Inventory: discover what Creo files exist
	var configFiles []string
	for _, name := range []string{"creo.yml", "creo.yaml", "creo.yml.tmp"} {
		path := filepath.Join(cwd, name)
		if _, err := os.Stat(path); err == nil {
			configFiles = append(configFiles, name)
		}
	}

	creoDir := filepath.Join(cwd, ".creo")
	hasCreoDir := false
	if fi, err := os.Stat(creoDir); err == nil && fi.IsDir() {
		hasCreoDir = true
	}

	// Nothing to drop
	if len(configFiles) == 0 && !hasCreoDir {
		fmt.Println(ui.SectionHeader("drop"))
		fmt.Println()
		ui.Info("No Creo files found in this project.")
		fmt.Println()
		return nil
	}

	// Check for active services
	process.CleanStale(cwd)
	reg, _ := process.LoadRegistry(cwd)

	activeServices := make(map[string]int)
	for name, pid := range reg {
		if process.IsPIDAlive(pid) {
			activeServices[name] = pid
		}
	}

	fmt.Println(ui.SectionHeader("drop"))
	fmt.Println()

	// --- Dry Run ---
	if dryRun {
		fmt.Println("  " + ui.Dim.Render("dry-run: no files will be removed"))
		fmt.Println()

		if len(activeServices) > 0 {
			fmt.Printf("  would stop %d active service(s):\n", len(activeServices))
			for name, pid := range activeServices {
				fmt.Printf("    • %s %s\n", name, ui.Dim.Render(fmt.Sprintf("(pid %d)", pid)))
			}
			fmt.Println()
		}

		fmt.Println("  would remove:")
		for _, f := range configFiles {
			fmt.Printf("    • %s\n", f)
		}
		if hasCreoDir {
			fmt.Println("    • .creo/")
		}
		fmt.Println()
		return nil
	}

	// --- Show summary before confirmation ---
	if len(activeServices) > 0 {
		fmt.Printf("  %d active service(s) will be stopped:\n", len(activeServices))
		for name, pid := range activeServices {
			fmt.Printf("    • %s %s\n", name, ui.Dim.Render(fmt.Sprintf("(pid %d)", pid)))
		}
		fmt.Println()
	}

	fmt.Println("  files to remove:")
	for _, f := range configFiles {
		fmt.Printf("    • %s\n", f)
	}
	if hasCreoDir {
		fmt.Println("    • .creo/")
	}
	fmt.Println()

	// --- Confirmation ---
	if !forceDrop {
		confirmed := ui.Confirm("Drop Creo from this project? This cannot be undone.", false)
		if !confirmed {
			fmt.Println()
			ui.Info("Aborted. No files were removed.")
			fmt.Println()
			return nil
		}
	}

	fmt.Println()

	// --- Animation ---
	ui.PrintBannerDissolve()

	// --- Stop active services ---
	stoppedCount := 0
	if len(activeServices) > 0 {
		killed := process.StopAllSilent(reg, cwd)
		stoppedCount = len(killed)
	}

	// Unlock registry if lock exists
	process.UnlockRegistry(cwd)

	// --- Remove config files ---
	var removedConfigs []string
	for _, name := range configFiles {
		path := filepath.Join(cwd, name)
		if err := os.Remove(path); err == nil || os.IsNotExist(err) {
			removedConfigs = append(removedConfigs, name)
		}
	}

	// --- Remove .creo/ directory ---
	removedDir := false
	if hasCreoDir {
		for attempt := 0; attempt < 5; attempt++ {
			if err := os.RemoveAll(creoDir); err == nil || os.IsNotExist(err) {
				removedDir = true
				break
			}
			time.Sleep(150 * time.Millisecond)
		}
	}

	// --- Print results ---
	fmt.Println()
	if stoppedCount > 0 {
		ui.Success(fmt.Sprintf("Stopped %d active service(s)", stoppedCount))
	}
	for _, name := range removedConfigs {
		ui.Success(fmt.Sprintf("Removed %s", name))
	}
	if removedDir {
		ui.Success("Removed .creo/")
	} else if hasCreoDir {
		ui.Warn("Could not remove .creo/ — files may be locked. Try deleting manually.")
	}

	fmt.Println()
	fmt.Println("  " + ui.Accent.Render("Creo has been dropped from this project."))
	fmt.Println()

	return nil
}
