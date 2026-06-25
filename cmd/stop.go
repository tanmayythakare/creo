package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/tanmayythakare/creo/internal/config"
	"github.com/tanmayythakare/creo/internal/process"
	"github.com/tanmayythakare/creo/internal/ui"
)

var stopCmd = &cobra.Command{
	Use:   "stop [service]",
	Short: "Stop all services or a specific one",
	RunE:  runStop,
}

func init() {
	rootCmd.AddCommand(stopCmd)
}

func runStop(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
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

	reg, err := process.LoadRegistry(cwd)
	if err != nil || len(reg) == 0 {
		fmt.Println(ui.SectionHeader("stop"))
		fmt.Println()
		fmt.Println("  nothing to stop")
		fmt.Println()
		return nil
	}

	fmt.Println(ui.SectionHeader("stop"))
	fmt.Println()

	nameCol := lipgloss.NewStyle().Width(24)
	statusCol := lipgloss.NewStyle().Width(10)
	pidCol := lipgloss.NewStyle().Width(10)

	if len(args) > 0 {
		// Stop specific service
		name := args[0]
		if _, ok := cfg.Services[name]; !ok {
			ui.Error(fmt.Sprintf("service '%s' not found. Available services:", name))
			for sName := range cfg.Services {
				fmt.Printf("  • %s\n", sName)
			}
			fmt.Println()
			return nil
		}

		killed, pid, err := process.StopService(reg, name, cwd)
		if err != nil {
			ui.Error(err.Error())
			return nil
		}
		if killed {
			fmt.Printf("  %s %s %s %s\n",
				ui.SymCheck,
				nameCol.Render(ui.Truncate(name, 24)),
				statusCol.Render("killed"),
				pidCol.Render(ui.Dim.Render(fmt.Sprintf("pid %d", pid))),
			)
		} else {
			fmt.Printf("  %s %s %s\n",
				ui.SymEmpty,
				nameCol.Render(ui.Truncate(name, 24)),
				ui.Dim.Render("already stopped"),
			)
		}
	} else {
		// We check if at least one service is running
		someRunning := false
		for name := range cfg.Services {
			_, running := process.IsRunning(reg, name)
			if running {
				someRunning = true
			}
		}

		if !someRunning {
			fmt.Println("  nothing to stop")
			fmt.Println()
			return nil
		}

		for name := range cfg.Services {
			killed, pid, err := process.StopService(reg, name, cwd)
			if err != nil {
				ui.Error(err.Error())
				continue
			}
			if killed {
				fmt.Printf("  %s %s %s %s\n",
					ui.SymCheck,
					nameCol.Render(ui.Truncate(name, 24)),
					statusCol.Render("killed"),
					pidCol.Render(ui.Dim.Render(fmt.Sprintf("pid %d", pid))),
				)
			} else {
				fmt.Printf("  %s %s %s\n",
					ui.SymEmpty,
					nameCol.Render(ui.Truncate(name, 24)),
					ui.Dim.Render("already stopped"),
				)
			}
		}
	}

	fmt.Println()
	if len(args) > 0 {
		fmt.Println("  service stopped")
	} else {
		fmt.Println("  all services stopped")
	}
	fmt.Println()

	return nil
}
