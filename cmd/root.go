package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tanmayythakare/creo/internal/bootstrap"
	"github.com/tanmayythakare/creo/internal/config"
	"github.com/tanmayythakare/creo/internal/process"
	"github.com/tanmayythakare/creo/internal/ui"
	"github.com/tanmayythakare/creo/internal/workflow"
	"golang.org/x/sys/windows"
)

var Verbose bool
var dryRun bool

var rootCmd = &cobra.Command{
	Use:               "creo [target]",
	Short:             "creo — developer workflow CLI",
	Long:              "creo eliminates project startup friction. One command starts all your services.",
	SilenceUsage:      true,
	PersistentPreRunE: runStartupChecks,
	RunE:              runRoot,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&Verbose, "verbose", false, "show verbose output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print what would launch, don't actually launch")
}

func Execute() {
	// Enable Virtual Terminal Processing on Windows console to support ANSI escape sequences
	stdout := windows.Handle(os.Stdout.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(stdout, &mode); err == nil {
		_ = windows.SetConsoleMode(stdout, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	}

	// Load config to dynamically register workflows/services/aliases as subcommands
	if cwd, err := os.Getwd(); err == nil {
		if cfg, err := config.Load(cwd); err == nil {
			// Register workflows
			for wfName := range cfg.Workflows {
				if !bootstrap.IsBuiltinCommand(wfName) {
					registerDynamicCmd(wfName)
				}
			}
			// Register services and their aliases
			for sName, svc := range cfg.Services {
				if !bootstrap.IsBuiltinCommand(sName) {
					registerDynamicCmd(sName)
				}
				if svc.Alias != "" && !bootstrap.IsBuiltinCommand(svc.Alias) {
					registerDynamicCmd(svc.Alias)
				}
			}
		}
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func registerDynamicCmd(name string) {
	// Check if already registered
	for _, c := range rootCmd.Commands() {
		if c.Name() == name {
			return
		}
	}

	sub := &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("Start workflow, service, or alias '%s'", name),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRoot(cmd, []string{name})
		},
	}
	rootCmd.AddCommand(sub)
}

func runRoot(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("✗ cannot get working directory: %w. Fix: Check directory permissions", err)
	}

	cfg, err := config.Load(cwd)
	if err != nil {
		if strings.Contains(err.Error(), "creo.yml not found") {
			// No creo.yml found, print command help
			_ = cmd.Help()
			return nil
		}
		return err
	}

	process.CleanOrphans(cwd, cfg)

	if err := process.LockRegistry(cwd); err != nil {
		ui.Error(fmt.Sprintf("%v", err))
		return nil
	}
	defer process.UnlockRegistry(cwd)

	// Determine which workflow/service/alias to run
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
		// Fallback: Check if it's a direct service name or service alias
		var matchedService string
		for sName, svc := range cfg.Services {
			if sName == workflowName {
				matchedService = sName
				break
			}
			if svc.Alias != "" && svc.Alias == workflowName {
				matchedService = sName
				break
			}
		}

		if matchedService != "" {
			services = []string{matchedService}
		} else {
			ui.Error(fmt.Sprintf("workflow, service, or alias '%s' not found. Available workflows:", workflowName))
			for name := range cfg.Workflows {
				fmt.Printf("  • %s\n", name)
			}
			fmt.Println()
			return nil
		}
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
