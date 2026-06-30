package bootstrap

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/tanmayythakare/creo/internal/config"
	"github.com/tanmayythakare/creo/internal/ui"
)

// RunWizard drives the full creo init interactive flow
// Returns a populated Config ready to be written to creo.yml
func RunWizard(detected []DetectedService) (*config.Config, error) {
	ui.Section("Project Setup")

	projectName := ui.Ask("Project name", "my-project")

	cfg := &config.Config{
		Project:  config.Project{Name: projectName},
		Services: make(map[string]config.Service),
	}

	// --- Services ---
	ui.Section("Services")

	// Initialize list with auto-detected services
	services := append([]DetectedService{}, detected...)

	for {
		fmt.Println("  Configure services:")
		if len(services) == 0 {
			fmt.Println("    (no services added yet)")
		} else {
			for i, s := range services {
				confSuffix := ""
				if s.Confidence == "low" {
					confSuffix = ui.Dim.Render(" (low confidence)")
				}
				fmt.Printf("    [%d] %s (path: %s) → %s%s\n", i+1, s.Name, s.Path, s.SuggestedCmd, confSuffix)
			}
		}
		fmt.Println()
		fmt.Println("  Choose an option:")
		fmt.Println("    [1] Add service")
		fmt.Println("    [2] Remove service")
		fmt.Println("    [3] Edit service")
		fmt.Println("    [4] Reset to auto-detected")
		fmt.Println("    [5] Done")
		fmt.Println()

		opt := ui.Ask("Option", "5")
		if opt == "5" || strings.ToLower(opt) == "done" {
			if len(services) == 0 {
				ui.Warn("You must add at least one service before finishing.")
				continue
			}
			break
		}

		switch opt {
		case "1", "add":
			var name string
			for {
				name = ui.Ask("  Service name", "")
				name = strings.TrimSpace(name)
				if name == "" {
					ui.Warn("Service name cannot be empty. Please try again.")
					continue
				}
				if len([]rune(name)) > 50 {
					ui.Warn("Service name cannot exceed 50 characters.")
					continue
				}
				badNameChars := []string{"..", "/", "\\", "*", "?"}
				hasBadName := false
				for _, bc := range badNameChars {
					if strings.Contains(name, bc) {
						hasBadName = true
						break
					}
				}
				if hasBadName {
					ui.Warn("Service name cannot contain '..', '/', '\\', '*', or '?'.")
					continue
				}
				if !isValidServiceName(name) {
					ui.Warn("Service name can only contain alphanumeric characters, underscores, and hyphens. Please try again.")
					continue
				}
				// YAML reserved words check
				reserved := map[string]bool{"yes": true, "no": true, "on": true, "off": true, "true": true, "false": true, "null": true}
				if reserved[strings.ToLower(name)] {
					ui.Warn("Service name cannot be a YAML reserved word (yes, no, on, off, true, false, null). Please try again.")
					continue
				}
				// P1: Check duplicate service name
				duplicate := false
				for _, s := range services {
					if s.Name == name {
						duplicate = true
						break
					}
				}
				if duplicate {
					ui.Warn(fmt.Sprintf("Service '%s' already exists. Choose a unique name.", name))
					continue
				}
				break
			}

			var path string
			for {
				path = ui.Ask("  Path (relative to repo root)", name)
				path = strings.TrimSpace(path)
				badPathChars := []string{"..", "*", "?"}
				hasBadPath := false
				for _, bc := range badPathChars {
					if strings.Contains(path, bc) {
						hasBadPath = true
						break
					}
				}
				if hasBadPath {
					ui.Warn("Service path cannot contain '..', '*', or '?'.")
					continue
				}
				fi, err := os.Stat(path)
				if os.IsNotExist(err) {
					ui.Warn(fmt.Sprintf("Path '%s' does not exist. Please try again.", path))
					continue
				}
				if err == nil && !fi.IsDir() {
					ui.Warn(fmt.Sprintf("Path '%s' is a file, not a directory. Please try again.", path))
					continue
				}
				break
			}

			var cmd string
			for {
				cmd = ui.Ask("  Start command", "")
				cmd = strings.TrimSpace(cmd)
				if cmd == "" {
					ui.Warn("Start command cannot be empty.")
					continue
				}
				if len([]rune(cmd)) > 200 {
					ui.Warn("Start command cannot exceed 200 characters.")
					continue
				}
				badCmdChars := []string{"&&", "||", "|", ";", ">", "<", "`"}
				hasBadCmd := false
				for _, bc := range badCmdChars {
					if strings.Contains(cmd, bc) {
						hasBadCmd = true
						break
					}
				}
				if hasBadCmd {
					ui.Warn("Start command cannot contain '&&', '||', '|', ';', '>', '<', or backticks.")
					continue
				}
				break
			}

			alias := ui.Ask("  Alias (optional)", "")

			services = append(services, DetectedService{
				Name:         name,
				Path:         path,
				Framework:    "Manual",
				SuggestedCmd: cmd,
				Alias:        alias,
			})
			ui.Success(fmt.Sprintf("Added %s", name))

		case "2", "remove":
			if len(services) == 0 {
				ui.Warn("No services to remove.")
				continue
			}
			idxStr := ui.Ask("  Service number to remove", "")
			idx, err := strconv.Atoi(idxStr)
			if err != nil || idx < 1 || idx > len(services) {
				ui.Warn("Invalid selection.")
				continue
			}
			removed := services[idx-1]
			services = append(services[:idx-1], services[idx:]...)
			ui.Success(fmt.Sprintf("Removed %s", removed.Name))

		case "3", "edit":
			if len(services) == 0 {
				ui.Warn("No services to edit.")
				continue
			}
			idxStr := ui.Ask("  Service number to edit", "")
			idx, err := strconv.Atoi(idxStr)
			if err != nil || idx < 1 || idx > len(services) {
				ui.Warn("Invalid selection.")
				continue
			}
			s := &services[idx-1]

			// Edit name
			for {
				name := ui.Ask(fmt.Sprintf("  Service name [%s]", s.Name), s.Name)
				name = strings.TrimSpace(name)
				if name == "" {
					ui.Warn("Service name cannot be empty.")
					continue
				}
				if len([]rune(name)) > 50 {
					ui.Warn("Service name cannot exceed 50 characters.")
					continue
				}
				badNameChars := []string{"..", "/", "\\", "*", "?"}
				hasBadName := false
				for _, bc := range badNameChars {
					if strings.Contains(name, bc) {
						hasBadName = true
						break
					}
				}
				if hasBadName {
					ui.Warn("Service name cannot contain '..', '/', '\\', '*', or '?'.")
					continue
				}
				if !isValidServiceName(name) {
					ui.Warn("Service name can only contain alphanumeric characters, underscores, and hyphens. Please try again.")
					continue
				}
				if name != s.Name {
					// P1: Check duplicate service name
					duplicate := false
					for i, existing := range services {
						if i != idx-1 && existing.Name == name {
							duplicate = true
							break
						}
					}
					if duplicate {
						ui.Warn(fmt.Sprintf("Service '%s' already exists. Choose a unique name.", name))
						continue
					}
				}
				s.Name = name
				break
			}

			// Edit path
			for {
				path := ui.Ask(fmt.Sprintf("  Path [%s]", s.Path), s.Path)
				path = strings.TrimSpace(path)
				badPathChars := []string{"..", "*", "?"}
				hasBadPath := false
				for _, bc := range badPathChars {
					if strings.Contains(path, bc) {
						hasBadPath = true
						break
					}
				}
				if hasBadPath {
					ui.Warn("Service path cannot contain '..', '*', or '?'.")
					continue
				}
				fi, err := os.Stat(path)
				if os.IsNotExist(err) {
					ui.Warn(fmt.Sprintf("Path '%s' does not exist. Please try again.", path))
					continue
				}
				if err == nil && !fi.IsDir() {
					ui.Warn(fmt.Sprintf("Path '%s' is a file, not a directory. Please try again.", path))
					continue
				}
				s.Path = path
				break
			}

			// Edit command
			for {
				cmd := ui.Ask(fmt.Sprintf("  Start command [%s]", s.SuggestedCmd), s.SuggestedCmd)
				cmd = strings.TrimSpace(cmd)
				if cmd == "" {
					ui.Warn("Start command cannot be empty.")
					continue
				}
				if len([]rune(cmd)) > 200 {
					ui.Warn("Start command cannot exceed 200 characters.")
					continue
				}
				badCmdChars := []string{"&&", "||", "|", ";", ">", "<", "`"}
				hasBadCmd := false
				for _, bc := range badCmdChars {
					if strings.Contains(cmd, bc) {
						hasBadCmd = true
						break
					}
				}
				if hasBadCmd {
					ui.Warn("Start command cannot contain '&&', '||', '|', ';', '>', '<', or backticks.")
					continue
				}
				s.SuggestedCmd = cmd
				break
			}

			s.Alias = ui.Ask(fmt.Sprintf("  Alias [%s]", s.Alias), s.Alias)
			ui.Success(fmt.Sprintf("Updated %s", s.Name))

		case "4", "reset", "restart":
			services = append([]DetectedService{}, detected...)
			ui.Success("Reset to initial auto-detected services.")

		default:
			ui.Warn("Invalid option.")
		}
		fmt.Println()
	}

	for _, s := range services {
		svc := config.Service{
			Path:    s.Path,
			Command: s.SuggestedCmd,
		}
		if s.Alias != "" {
			svc.Alias = s.Alias
		}
		cfg.Services[s.Name] = svc
	}

	var serviceNames []string
	for name := range cfg.Services {
		serviceNames = append(serviceNames, name)
	}

	// --- Workflows ---
	ui.Section("Workflows")

	fmt.Println("  A workflow is a named group of services to start together.")
	fmt.Printf("  Available services: %s\n\n", strings.Join(serviceNames, ", "))

	cfg.Workflows = make(map[string][]string)

	// Default workflow name prompt - re-ask if empty or spaces
	var defaultWorkflow string
	for {
		defaultWorkflow = ui.Ask("Default workflow name (e.g. boom, start, up)", "")
		defaultWorkflow = strings.TrimSpace(defaultWorkflow)
		if defaultWorkflow == "" {
			ui.Warn("Workflow name cannot be empty or only spaces. Please try again.")
			continue
		}
		if IsBuiltinCommand(defaultWorkflow) {
			ui.Warn(fmt.Sprintf("Workflow name '%s' conflicts with built-in command. Please try again.", defaultWorkflow))
			continue
		}
		break
	}

	cfg.Workflows[defaultWorkflow] = serviceNames
	ui.Success(fmt.Sprintf("Workflow '%s' → %s", defaultWorkflow, strings.Join(serviceNames, ", ")))

	// Allow adding more workflows
	for {
		more := ui.Confirm("Add another workflow?", false)
		if !more {
			break
		}

		var wfName string
		for {
			wfName = ui.Ask("  Workflow name", "")
			wfName = strings.TrimSpace(wfName)
			if wfName == "" {
				ui.Warn("Workflow name cannot be empty or only spaces. Please try again.")
				continue
			}
			if strings.Contains(wfName, " ") {
				wfName = strings.ReplaceAll(wfName, " ", "-")
				ui.Info(fmt.Sprintf("  Spaces converted to hyphens: '%s'", wfName))
			}
			if IsBuiltinCommand(wfName) {
				ui.Warn(fmt.Sprintf("Workflow name '%s' conflicts with built-in command. Please try again.", wfName))
				continue
			}
			break
		}

		fmt.Printf("  Available services: %s\n", strings.Join(serviceNames, ", "))
		var selected []string
		for {
			raw := ui.Ask("  Services (comma-separated)", strings.Join(serviceNames, ","))
			selected = nil
			var invalidServices []string

			for _, s := range strings.Split(raw, ",") {
				s = strings.TrimSpace(s)
				if s != "" {
					exists := false
					for _, existing := range serviceNames {
						if existing == s {
							exists = true
							break
						}
					}
					if exists {
						selected = append(selected, s)
					} else {
						invalidServices = append(invalidServices, s)
					}
				}
			}

			if len(invalidServices) > 0 {
				ui.Warn(fmt.Sprintf("The following service(s) do not exist: %s. Please enter only valid services.", strings.Join(invalidServices, ", ")))
				continue
			}
			if len(selected) == 0 {
				ui.Warn("Workflow must contain at least one service. Please try again.")
				continue
			}
			break
		}

		cfg.Workflows[wfName] = selected
		ui.Success(fmt.Sprintf("Workflow '%s' → %s", wfName, strings.Join(selected, ", ")))
	}

	return cfg, nil
}

func isValidServiceName(name string) bool {
	if len(name) == 0 {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.') {
			return false
		}
	}
	return true
}

func IsBuiltinCommand(name string) bool {
	builtins := map[string]bool{
		"init":    true,
		"up":      true,
		"status":  true,
		"stop":    true,
		"restart": true,
		"doctor":  true,
		"version": true,
		"help":    true,
	}
	return builtins[strings.ToLower(name)]
}
