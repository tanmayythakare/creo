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

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show running status of all services",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	cfg, err := config.Load(cwd)
	if err != nil {
		ui.Error(err.Error())
		return nil
	}

	reg, _ := process.LoadRegistry(cwd)

	fmt.Println(ui.SectionHeader("status"))
	fmt.Println()

	maxNameLen := 12
	maxCmdLen := 20
	for name, svc := range cfg.Services {
		if len(name) > maxNameLen {
			maxNameLen = len(name)
		}
		if len(svc.Command) > maxCmdLen {
			maxCmdLen = len(svc.Command)
		}
	}
	if maxNameLen > 24 {
		maxNameLen = 24
	}
	if maxCmdLen > 30 {
		maxCmdLen = 30
	}

	nameCol := lipgloss.NewStyle().Width(maxNameLen)
	cmdCol := lipgloss.NewStyle().Width(maxCmdLen)
	statusCol := lipgloss.NewStyle().Width(12)

	for name, svc := range cfg.Services {
		pid, running := process.IsRunning(reg, name)

		sym := ui.SymEmpty
		statusText := ui.Dim.Render("stopped")
		if running {
			sym = ui.SymDot
			statusText = ui.Dim.Render(fmt.Sprintf("pid %d", pid))
		}

		fmt.Printf("  %s %s %s %s\n",
			sym,
			nameCol.Render(ui.Truncate(name, maxNameLen)),
			cmdCol.Render(ui.Truncate(svc.Command, maxCmdLen)),
			statusCol.Render(statusText),
		)
	}

	fmt.Println()
	return nil
}
