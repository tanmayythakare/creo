package doctor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/tanmayythakare/creo/internal/config"
)

// CheckResult holds the result of a single tool check
type CheckResult struct {
	Tool        string
	Available   bool
	Version     string
	InstallHint string
	Warning     string
}

// ServiceReport holds all check results for one service
type ServiceReport struct {
	ServiceName string
	Path        string
	Command     string
	Checks      []CheckResult
}

// SystemReport is the full doctor output
type SystemReport struct {
	WTAvailable bool // Windows Terminal
	WTVersion   string
	WTWarning   string
	Services    []ServiceReport
	Warnings    []string
	AllOK       bool
}

// Run performs all checks and returns a SystemReport
func Run(cfg *config.Config) SystemReport {
	report := SystemReport{}

	// Check Windows Terminal
	report.WTAvailable = commandExists("wt.exe")
	if report.WTAvailable {
		report.WTVersion, report.WTWarning = CheckWTVersion()
	}

	// Environment warnings (WSL, Docker, OneDrive)
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		report.Warnings = append(report.Warnings, "Running inside WSL. Creo is designed for native Windows Terminal with wt.exe.")
	}
	if _, err := os.Stat("/.dockerenv"); err == nil {
		report.Warnings = append(report.Warnings, "Running inside a Docker container. Creo requires a native Windows environment to launch wt.exe.")
	}
	cwd, err := os.Getwd()
	if err == nil {
		if strings.Contains(strings.ToLower(cwd), "onedrive") {
			report.Warnings = append(report.Warnings, "Project is inside a OneDrive folder. OneDrive file syncing can lock PID/log files and cause race conditions.")
		}
	}

	allOK := report.WTAvailable && report.WTWarning == ""

	if cfg != nil {
		for name, svc := range cfg.Services {
			sr := ServiceReport{
				ServiceName: name,
				Path:        svc.Path,
				Command:     svc.Command,
			}

			tools := inferRequiredTools(svc.Command)
			for _, tool := range tools {
				result := checkTool(tool)
				if tool == "java" && result.Available {
					result.Warning = checkJavaVersion(result.Version)
				}
				sr.Checks = append(sr.Checks, result)
				if !result.Available || result.Warning != "" {
					allOK = false
				}
			}

			report.Services = append(report.Services, sr)
		}
	}

	report.AllOK = allOK
	return report
}

// inferRequiredTools maps a command string to the tools needed to run it
func inferRequiredTools(command string) []string {
	cmd := strings.ToLower(strings.TrimSpace(command))

	switch {
	case strings.HasPrefix(cmd, "npm"):
		return []string{"node", "npm"}
	case strings.HasPrefix(cmd, "npx"):
		return []string{"node", "npx"}
	case strings.HasPrefix(cmd, "ng "):
		return []string{"node", "npm"}
	case strings.HasPrefix(cmd, "mvn"):
		return []string{"java", "mvn"}
	case strings.HasPrefix(cmd, "./gradlew") || strings.HasPrefix(cmd, "gradle"):
		return []string{"java", "gradle"}
	case strings.HasPrefix(cmd, "go "):
		return []string{"go"}
	case strings.HasPrefix(cmd, "docker compose") || strings.HasPrefix(cmd, "docker-compose"):
		return []string{"docker"}
	case strings.HasPrefix(cmd, "docker"):
		return []string{"docker"}
	case strings.HasPrefix(cmd, "python") || strings.HasPrefix(cmd, "python3"):
		return []string{"python"}
	case strings.HasPrefix(cmd, "cargo"):
		return []string{"cargo"}
	case strings.HasPrefix(cmd, "php"):
		return []string{"php"}
	case strings.HasPrefix(cmd, "ruby") || strings.HasPrefix(cmd, "bundle"):
		return []string{"ruby"}
	case strings.HasPrefix(cmd, "flutter"):
		return []string{"flutter"}
	case strings.HasPrefix(cmd, "dotnet"):
		return []string{"dotnet"}
	case strings.HasPrefix(cmd, "deno"):
		return []string{"deno"}
	case strings.HasPrefix(cmd, "bun"):
		return []string{"bun"}
	default:
		// Try to extract the first word as the binary, only if it's on our strict allowlist
		parts := strings.Fields(cmd)
		if len(parts) > 0 {
			binary := parts[0]
			allowlist := map[string]bool{
				"go":      true,
				"node":    true,
				"java":    true,
				"python":  true,
				"python3": true,
				"npm":     true,
				"npx":     true,
				"cargo":   true,
				"docker":  true,
				"dotnet":  true,
				"mvn":     true,
				"gradle":  true,
				"php":     true,
				"ruby":    true,
				"flutter": true,
				"deno":    true,
				"bun":     true,
			}
			if allowlist[binary] {
				return []string{binary}
			}
		}
		return nil
	}
}

// checkTool checks if a tool binary exists and gets its version
func checkTool(tool string) CheckResult {
	result := CheckResult{
		Tool:        tool,
		InstallHint: installHint(tool),
	}

	if !commandExists(tool) {
		result.Available = false
		return result
	}

	result.Available = true
	result.Version = getVersion(tool)
	return result
}

// commandExists checks if a binary is in PATH
func commandExists(binary string) bool {
	_, err := exec.LookPath(binary)
	return err == nil
}

// getVersion tries to get a short version string for known tools
func getVersion(tool string) string {
	versionArgs := map[string][]string{
		"node":    {"--version"},
		"npm":     {"--version"},
		"npx":     {"--version"},
		"java":    {"-version"},
		"mvn":     {"--version"},
		"go":      {"version"},
		"docker":  {"--version"},
		"python":  {"--version"},
		"python3": {"--version"},
		"cargo":   {"--version"},
		"php":     {"--version"},
		"ruby":    {"--version"},
		"flutter": {"--version"},
		"dotnet":  {"--version"},
		"deno":    {"--version"},
		"bun":     {"--version"},
		"gradle":  {"--version"},
	}

	args, ok := versionArgs[tool]
	if !ok {
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, tool, args...).CombinedOutput()
	if err != nil {
		return "corrupted (returned error)"
	}

	// Return first line only, trimmed
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) > 0 {
		v := strings.TrimSpace(lines[0])
		// Trim common prefixes for cleaner output
		v = strings.TrimPrefix(v, "v")
		if len(v) > 40 {
			v = v[:40] + "..."
		}
		return v
	}
	return ""
}

// installHint returns a Windows-friendly install suggestion
func installHint(tool string) string {
	hints := map[string]string{
		"node":    "winget install OpenJS.NodeJS  OR  https://nodejs.org",
		"npm":     "comes with Node.js — install node first",
		"npx":     "comes with Node.js — install node first",
		"java":    "winget install Microsoft.OpenJDK.21  OR  https://adoptium.net",
		"mvn":     "winget install Apache.Maven  OR  https://maven.apache.org/download.cgi",
		"go":      "winget install GoLang.Go  OR  https://go.dev/dl",
		"docker":  "winget install Docker.DockerDesktop  OR  https://www.docker.com/products/docker-desktop",
		"python":  "winget install Python.Python.3  OR  https://python.org",
		"python3": "winget install Python.Python.3  OR  https://python.org",
		"cargo":   "winget install Rustlang.Rust.MSVC  OR  https://rustup.rs",
		"php":     "winget install PHP.PHP  OR  https://windows.php.net",
		"ruby":    "winget install RubyInstallerTeam.Ruby  OR  https://rubyinstaller.org",
		"wt.exe":  "Install Windows Terminal from Microsoft Store  OR  winget install Microsoft.WindowsTerminal",
		"flutter": "winget install Google.Flutter",
		"dotnet":  "winget install Microsoft.DotNet.SDK.8",
		"deno":    "winget install DenoLand.Deno",
		"bun":     "winget install Oven-sh.Bun",
		"gradle":  "winget install Gradle.Gradle",
	}

	if hint, ok := hints[tool]; ok {
		return hint
	}
	return fmt.Sprintf("Install '%s' and make sure it's in your PATH", tool)
}

func CheckWTVersion() (string, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", "Get-AppxPackage -Name Microsoft.WindowsTerminal | Select-Object -ExpandProperty Version")
	out, err := cmd.Output()
	if err != nil {
		return "unknown", ""
	}
	v := strings.TrimSpace(string(out))
	if v == "" {
		return "unknown", ""
	}
	// Take only the first line in case there are multiple installations
	if idx := strings.Index(v, "\n"); idx != -1 {
		v = strings.TrimSpace(v[:idx])
	}
	parts := strings.Split(v, ".")
	if len(parts) >= 2 {
		major, _ := strconv.Atoi(parts[0])
		minor, _ := strconv.Atoi(parts[1])
		if major < 1 || (major == 1 && minor < 12) {
			return v, fmt.Sprintf("Windows Terminal version %s is old. Version 1.12+ is required.", v)
		}
	}
	return v, ""
}

func checkJavaVersion(versionStr string) string {
	v := strings.ToLower(versionStr)
	if strings.Contains(v, "corrupted") || v == "" {
		return ""
	}

	var verNum string
	if idx := strings.Index(v, "version"); idx != -1 {
		sub := v[idx+7:]
		if qIdx := strings.Index(sub, "\""); qIdx != -1 {
			sub = sub[qIdx+1:]
			if endQ := strings.Index(sub, "\""); endQ != -1 {
				verNum = sub[:endQ]
			}
		}
	}
	if verNum == "" {
		fields := strings.Fields(v)
		for _, f := range fields {
			f = strings.Trim(f, "\"`'")
			if len(f) > 0 && (f[0] >= '0' && f[0] <= '9') {
				verNum = f
				break
			}
		}
	}

	if verNum != "" {
		parts := strings.Split(verNum, ".")
		if len(parts) > 0 {
			major, err := strconv.Atoi(parts[0])
			if err == nil {
				if major == 1 && len(parts) > 1 {
					major, _ = strconv.Atoi(parts[1])
				}
				if major < 17 {
					return fmt.Sprintf("Java version %s detected. Spring Boot requires Java 17+.", verNum)
				}
			}
		}
	}
	return ""
}
