package doctor

import (
	"strings"
	"testing"

	"github.com/tanmayythakare/creo/internal/config"
)

func TestInferRequiredTools(t *testing.T) {
	tests := []struct {
		cmd      string
		expected []string
	}{
		{"npm run dev", []string{"node", "npm"}},
		{"npx tailwindcss", []string{"node", "npx"}},
		{"ng serve", []string{"node", "npm"}},
		{"mvn spring-boot:run", []string{"java", "mvn"}},
		{"./gradlew bootRun", []string{"java", "gradle"}},
		{"go run main.go", []string{"go"}},
		{"docker compose up", []string{"docker"}},
		{"python app.py", []string{"python"}},
		{"cargo build", []string{"cargo"}},
		{"php artisan serve", []string{"php"}},
		{"ruby app.rb", []string{"ruby"}},
		{"flutter run", []string{"flutter"}},
		{"dotnet run", []string{"dotnet"}},
		{"deno run app.ts", []string{"deno"}},
		{"bun run app.js", []string{"bun"}},
		{"go build", []string{"go"}},
		{"custom_binary -v", nil}, // Not on strict allowlist
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			got := inferRequiredTools(tt.cmd)
			if len(got) != len(tt.expected) {
				t.Fatalf("expected %v, got %v", tt.expected, got)
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("at index %d: expected %s, got %s", i, tt.expected[i], got[i])
				}
			}
		})
	}
}

func TestCheckJavaVersion(t *testing.T) {
	tests := []struct {
		name       string
		versionStr string
		hasWarning bool
	}{
		{"Java 17 (success)", `openjdk version "17.0.8" 2023-07-18`, false},
		{"Java 21 (success)", `java version "21.0.1" 2023-10-17`, false},
		{"Java 11 (warning)", `openjdk version "11.0.12" 2021-07-20`, true},
		{"Java 8 (warning)", `java version "1.8.0_291"`, true},
		{"Corrupted", "corrupted (returned error)", false},
		{"Empty version", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkJavaVersion(tt.versionStr)
			if tt.hasWarning && got == "" {
				t.Error("expected version warning, got empty string")
			}
			if !tt.hasWarning && got != "" {
				t.Errorf("expected no warning, got: %s", got)
			}
		})
	}
}

func TestInstallHint(t *testing.T) {
	tests := []struct {
		tool           string
		containsSubstring string
	}{
		{"node", "winget install OpenJS.NodeJS"},
		{"java", "winget install Microsoft.OpenJDK"},
		{"mvn", "Apache.Maven"},
		{"wt.exe", "Microsoft.WindowsTerminal"},
		{"unknown_tool", "Install 'unknown_tool'"},
	}

	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			got := installHint(tt.tool)
			if !strings.Contains(got, tt.containsSubstring) {
				t.Errorf("expected hint to contain %q, got: %s", tt.containsSubstring, got)
			}
		})
	}
}

func TestCommandExists(t *testing.T) {
	t.Run("known command exists", func(t *testing.T) {
		// cmd.exe is guaranteed to exist on Windows
		if !commandExists("cmd.exe") {
			t.Error("expected cmd.exe to exist in PATH")
		}
	})

	t.Run("non-existent command", func(t *testing.T) {
		if commandExists("faketoolxyz12345.exe") {
			t.Error("expected non-existent tool to not exist")
		}
	})
}

func TestCheckWTVersion(t *testing.T) {
	// Call CheckWTVersion to ensure it executes without crashing on the host.
	// Since we are running on a Windows system, it might succeed or fail depending
	// on the execution policy, but it should return clean status values.
	_, _ = CheckWTVersion()
}

func TestRun(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		report := Run(nil)
		// Should not crash and should check WT availability
		if report.Warnings == nil {
			report.Warnings = []string{}
		}
	})

	t.Run("valid config checks", func(t *testing.T) {
		cfg := &config.Config{
			Project: config.Project{Name: "test-run"},
			Services: map[string]config.Service{
				"frontend": {
					Command: "npm run dev",
				},
				"backend": {
					Command: "go run main.go",
				},
			},
		}

		report := Run(cfg)
		if len(report.Services) != 2 {
			t.Errorf("expected 2 services in report, got %d", len(report.Services))
		}

		// Ensure tools were parsed and matched
		var foundFrontend bool
		for _, s := range report.Services {
			if s.ServiceName == "frontend" {
				foundFrontend = true
				if len(s.Checks) == 0 {
					t.Error("expected check results for frontend service")
				}
			}
		}
		if !foundFrontend {
			t.Error("frontend service not checked")
		}
	})
}
