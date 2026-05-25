package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tanmaythakare/creo/internal/config"
	"github.com/tanmaythakare/creo/internal/process"
	"github.com/tanmaythakare/creo/internal/ui"
	"github.com/tanmaythakare/creo/internal/workflow"
)

var dryRun bool

var upCmd = &cobra.Command{
	Use:   "up [workflow]",
	Short: "Start a workflow (default: first workflow in creo.yml)",
	RunE:  runUp,
}

func init() {
	rootCmd.AddCommand(upCmd)
	upCmd.Flags().BoolVar(&dryRun, "dry-run", false, "print what would launch, don't actually launch")
}

func runUp(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("✗ cannot get working directory: %w. Fix: Check directory permissions", err)
	}

	cfg, err := config.Load(cwd)
	if err != nil {
		ui.Error("No creo.yml found. Run: creo init")
		return nil
	}

	process.CleanOrphans(cwd, cfg)

	if err := process.LockRegistry(cwd); err != nil {
		ui.Error(fmt.Sprintf("%v", err))
		return nil
	}
	defer process.UnlockRegistry(cwd)

	// Determine which workflow to run
	workflowName := ""
	if len(args) > 0 {
		workflowName = args[0]
	} else {
		// Default to first workflow defined (sorted alphabetically for determinism)
		if len(cfg.Workflows) > 0 {
			names := make([]string, 0, len(cfg.Workflows))
			for name := range cfg.Workflows {
				names = append(names, name)
			}
			sort.Strings(names)
			workflowName = names[0]
		}
	}

	if workflowName == "" {
		ui.Error("✗ No workflows defined in creo.yml. Fix: Add a workflow to creo.yml")
		return nil
	}

	services, ok := cfg.Workflows[workflowName]
	if !ok {
		ui.Error(fmt.Sprintf("workflow '%s' not found. Available workflows:", workflowName))
		for name := range cfg.Workflows {
			fmt.Printf("  • %s\n", name)
		}
		fmt.Println()
		return nil
	}

	if len(services) == 0 {
		ui.Warn(fmt.Sprintf("workflow '%s' has 0 services. Nothing to launch.", workflowName))
		return nil
	}

	// Validate none of the services have empty commands
	for _, sName := range services {
		if svc, exists := cfg.Services[sName]; exists {
			if strings.TrimSpace(svc.Command) == "" {
				ui.Error(fmt.Sprintf("service '%s' has an empty command. Start rejected.", sName))
				return nil
			}
		}
	}

	return workflow.Start(cfg, workflowName, services, cwd, dryRun, Verbose)
}
