package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tanmayythakare/creo/internal/config"
	"github.com/tanmayythakare/creo/internal/process"
	"github.com/tanmayythakare/creo/internal/ui"
	"github.com/tanmayythakare/creo/internal/workflow"
)

var restartCmd = &cobra.Command{
	Use:   "restart <service>",
	Short: "Restart a specific service",
	Args:  cobra.ExactArgs(1),
	RunE:  runRestart,
}

func init() {
	rootCmd.AddCommand(restartCmd)
}

func runRestart(cmd *cobra.Command, args []string) error {
	name := args[0]

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	cfg, err := config.Load(cwd)
	if err != nil {
		ui.Error(err.Error())
		return nil
	}

	process.CleanOrphans(cwd, cfg)

	if err := process.LockRegistry(cwd); err != nil {
		ui.Error(fmt.Sprintf("%v", err))
		return nil
	}
	defer process.UnlockRegistry(cwd)

	svc, ok := cfg.Services[name]
	if !ok {
		ui.Error(fmt.Sprintf("service '%s' not found. Available services:", name))
		for sName := range cfg.Services {
			fmt.Printf("  • %s\n", sName)
		}
		fmt.Println()
		return nil
	}

	if strings.TrimSpace(svc.Command) == "" {
		ui.Error(fmt.Sprintf("service '%s' has an empty command. Restart rejected.", name))
		return nil
	}

	fmt.Println(ui.SectionHeader(fmt.Sprintf("restart %s", name)))
	fmt.Println()

	reg, err := process.LoadRegistry(cwd)
	var oldPID int
	var hadPID bool
	if err == nil {
		if pid, exists := reg[name]; exists && pid != 0 {
			oldPID = pid
			hadPID = true
		}
		// Stop if running
		killed, _, errStop := process.StopService(reg, name, cwd)
		if errStop != nil {
			ui.Error(errStop.Error())
			return nil
		}
		if killed {
			fmt.Printf("  %s %s   %s\n", ui.SymCheck, name, ui.Dim.Render("stopped"))
		}
	}

	svcPath := svc.Path
	if svcPath == "" {
		svcPath = "."
	}
	absPath := filepath.Join(cwd, svcPath)
	svc = workflow.ResolveCommand(name, svc, absPath, cwd)

	var launchErr error
	ui.SpinUntil("relaunching "+name, func() {
		launchErr = workflow.LaunchTab(name, svc, cwd, Verbose)
	})

	if launchErr != nil {
		ui.Error(fmt.Sprintf("failed to relaunch %s: %v", name, launchErr))
		if hadPID {
			_ = process.SetPID(cwd, name, oldPID)
		}
		return nil
	}

	fmt.Println()
	fmt.Println("  done")
	fmt.Println()

	return nil
}
