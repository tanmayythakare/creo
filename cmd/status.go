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
		ui.Error("No creo.yml found. Run: creo init")
		return nil
	}

	reg, _ := process.LoadRegistry(cwd)

	fmt.Println(ui.SectionHeader("status"))
	fmt.Println()

	nameCol := lipgloss.NewStyle().Width(24)
	cmdCol := lipgloss.NewStyle().Width(24)
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
			nameCol.Render(ui.Truncate(name, 24)),
			cmdCol.Render(ui.Truncate(svc.Command, 24)),
			statusCol.Render(statusText),
		)
	}

	fmt.Println()
	return nil
}
