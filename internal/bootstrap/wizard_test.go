package bootstrap

import (
	"os"
	"strings"
	"testing"

	"github.com/tanmayythakare/creo/internal/ui"
)

func TestIsValidServiceName(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"frontend", true},
		{"web-app", true},
		{"db_service", true},
		{"service123", true},
		{"", false},
		{"next.js", true},
		{"web/app", false},
		{"app*srv", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidServiceName(tt.name)
			if got != tt.expected {
				t.Errorf("isValidServiceName(%q) expected %v, got %v", tt.name, tt.expected, got)
			}
		})
	}
}

func TestIsBuiltinCommand(t *testing.T) {
	tests := []struct {
		cmd      string
		expected bool
	}{
		{"init", true},
		{"up", true},
		{"status", true},
		{"stop", true},
		{"restart", true},
		{"doctor", true},
		{"version", true},
		{"help", true},
		{"drop", true},
		{"delete", true},
		{"DROP", true},
		{"DELETE", true},
		{"INIT", true},
		{"my-workflow", false},
		{"dev", false},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			got := IsBuiltinCommand(tt.cmd)
			if got != tt.expected {
				t.Errorf("IsBuiltinCommand(%q) expected %v, got %v", tt.cmd, tt.expected, got)
			}
		})
	}
}

func TestRunWizard(t *testing.T) {
	tempDir := t.TempDir()

	// Move to tempDir so that path validation (os.Stat) in RunWizard works for "."
	oldCwd, err := os.Getwd()
	if err == nil {
		defer func() { _ = os.Chdir(oldCwd) }()
		_ = os.Chdir(tempDir)
	}

	t.Run("basic interactive setup", func(t *testing.T) {
		// Mock responses:
		// 1. Project name: "interactive-test"
		// 2. Menu option: "5" (Done) -> Fail because no services yet
		// 3. Menu option: "1" (Add)
		//    - Service name: "api"
		//    - Path: "."
		//    - Start command: "go run main.go"
		//    - Alias: "backend"
		// 4. Menu option: "5" (Done)
		// 5. Default workflow name: "boom"
		// 6. Add another workflow: "n" (No)
		inputs := []string{
			"interactive-test",
			"5", // attempt done with no services
			"1", // add service
			"api",
			".",
			"go run main.go",
			"backend",
			"5", // done with one service
			"boom",
			"n", // no more workflows
		}

		// Inject mock scanner reader
		ui.SetReader(strings.NewReader(strings.Join(inputs, "\n") + "\n"))

		detected := []DetectedService{} // empty detection
		cfg, err := RunWizard(detected)
		if err != nil {
			t.Fatalf("wizard failed: %v", err)
		}

		if cfg.Project.Name != "interactive-test" {
			t.Errorf("expected project interactive-test, got %s", cfg.Project.Name)
		}

		svc, ok := cfg.Services["api"]
		if !ok {
			t.Fatal("expected service 'api' to be created")
		}
		if svc.Path != "." || svc.Command != "go run main.go" || svc.Alias != "backend" {
			t.Errorf("service 'api' not created correctly: %+v", svc)
		}

		wf, ok := cfg.Workflows["boom"]
		if !ok || len(wf) != 1 || wf[0] != "api" {
			t.Errorf("workflow 'boom' not created correctly: %v", wf)
		}
	})

	t.Run("setup with auto-detected services", func(t *testing.T) {
		// Mock responses:
		// 1. Project name: "detected-test"
		// 2. Menu option: "5" (Done - accept detected)
		// 3. Default workflow name: "up"
		// 4. Add another workflow: "n" (No)
		inputs := []string{
			"detected-test",
			"5",
			"dev",
			"n",
		}

		ui.SetReader(strings.NewReader(strings.Join(inputs, "\n") + "\n"))

		detected := []DetectedService{
			{
				Name:         "frontend",
				Path:         ".",
				SuggestedCmd: "npm run dev",
				Framework:    "Next.js",
			},
		}

		cfg, err := RunWizard(detected)
		if err != nil {
			t.Fatalf("wizard failed: %v", err)
		}

		if cfg.Project.Name != "detected-test" {
			t.Errorf("expected project detected-test, got %s", cfg.Project.Name)
		}

		svc, ok := cfg.Services["frontend"]
		if !ok {
			t.Fatal("expected auto-detected service 'frontend' to be registered")
		}
		if svc.Command != "npm run dev" {
			t.Errorf("expected 'npm run dev', got '%s'", svc.Command)
		}
	})
}
