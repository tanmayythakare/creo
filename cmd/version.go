package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanmayythakare/creo/internal/ui"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print creo version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("  %s  %s\n", ui.Accent.Render("creo"), ui.Dim.Render(ui.Version))
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
