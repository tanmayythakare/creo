package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tanmaythakare/creo/internal/ui"
	"golang.org/x/sys/windows"
)

var Verbose bool

var rootCmd = &cobra.Command{
	Use:   "creo",
	Short: "creo — developer workflow CLI",
	Long:  "creo eliminates project startup friction. One command starts all your services.",
	// Don't show usage on errors — keep output clean
	SilenceUsage:      true,
	PersistentPreRunE: runStartupChecks,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&Verbose, "verbose", false, "show verbose output")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runStartupChecks(cmd *cobra.Command, args []string) error {
	// Skip check for version or help commands
	if cmd.Name() == "version" || cmd.Name() == "help" {
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}

	// 1. Check UNC path
	if strings.HasPrefix(cwd, `\\`) {
		fmt.Println()
		ui.Warn("⚠ creo detected a network path (" + cwd + ")\n  Windows Terminal may not support this location.")
		choice := ui.Ask("Continue anyway? [y/N]", "n")
		choice = strings.ToLower(strings.TrimSpace(choice))
		if choice != "y" && choice != "yes" {
			fmt.Println("  Aborting. Exiting cleanly.")
			os.Exit(0)
		}
	}

	// 2. Check OneDrive
	if strings.Contains(strings.ToLower(cwd), "onedrive") {
		fmt.Println()
		ui.Warn("⚠ creo detected a OneDrive or synced folder.\n  File sync may cause conflicts with .creo/pids.json")
		choice := ui.Ask("Continue anyway? [y/N]", "n")
		choice = strings.ToLower(strings.TrimSpace(choice))
		if choice != "y" && choice != "yes" {
			fmt.Println("  Aborting. Exiting cleanly.")
			os.Exit(0)
		}
	}

	// 3. Check Running as Administrator
	if isRunningAsAdmin() {
		fmt.Println()
		ui.Warn("⚠ creo is running as Administrator.\n  Services launched may behave differently than expected.")
		choice := ui.Ask("Continue anyway? [y/N]", "n")
		choice = strings.ToLower(strings.TrimSpace(choice))
		if choice != "y" && choice != "yes" {
			fmt.Println("  Aborting. Exiting cleanly.")
			os.Exit(0)
		}
	}

	// 4. Check Disk Space Low
	if freeSpace, err := getFreeDiskSpace(cwd); err == nil {
		limit := uint64(100 * 1024 * 1024) // 100MB
		if freeSpace < limit {
			mb := freeSpace / (1024 * 1024)
			fmt.Println()
			ui.Warn(fmt.Sprintf("⚠ Low disk space detected (~%dmb remaining).\n  creo may fail to write config or PID files.", mb))
			choice := ui.Ask("Continue anyway? [y/N]", "n")
			choice = strings.ToLower(strings.TrimSpace(choice))
			if choice != "y" && choice != "yes" {
				fmt.Println("  Aborting. Exiting cleanly.")
				os.Exit(0)
			}
		}
	}

	return nil
}

func isRunningAsAdmin() bool {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid,
	)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	var token windows.Token
	err = windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token)
	if err != nil {
		return false
	}
	defer token.Close()

	member, err := token.IsMember(sid)
	if err != nil {
		return false
	}
	return member
}

func getFreeDiskSpace(path string) (uint64, error) {
	var freeBytesAvailable uint64
	var totalNumberOfBytes uint64
	var totalNumberOfFreeBytes uint64

	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}

	err = windows.GetDiskFreeSpaceEx(pathPtr, &freeBytesAvailable, &totalNumberOfBytes, &totalNumberOfFreeBytes)
	if err != nil {
		return 0, err
	}
	return freeBytesAvailable, nil
}
