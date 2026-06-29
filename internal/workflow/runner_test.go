package workflow

import (
	"strings"
	"testing"

	"github.com/tanmayythakare/creo/internal/config"
)

func TestBuildWTArgs(t *testing.T) {
	entries := []serviceEntry{
		{
			name: "app",
			svc: config.Service{
				Path:    "frontend",
				Command: `set NODE_ENV=dev; npm run dev --prefix "my folder"`,
			},
		},
		{
			name: "api",
			svc: config.Service{
				Path:    ".",
				Command: "go run main.go",
			},
		},
	}

	root := "C:\\Projects\\creo-test"
	args := buildWTArgs(entries, root)

	// Join to easily check contents
	argsStr := strings.Join(args, " ")

	// Test 1: Semicolons should be escaped
	if !strings.Contains(argsStr, `\;`) {
		t.Errorf("Expected semicolons to be escaped as \\;, got: %s", argsStr)
	}

	// Test 2: Ensure quotes are NOT manually escaped with backslashes
	if strings.Contains(argsStr, `\"my folder\"`) {
		t.Errorf("Expected quotes to be preserved without backslashes for os/exec, got: %s", argsStr)
	}

	// Test 3: Ensure -d flag is used for starting directory
	if !strings.Contains(argsStr, `-d C:\Projects\creo-test\frontend`) {
		t.Errorf("Expected -d flag for starting directory, got: %s", argsStr)
	}

	// Test 4: Ensure cd /d is removed
	if strings.Contains(argsStr, `cd /d`) {
		t.Errorf("Expected 'cd /d' to be removed from command, got: %s", argsStr)
	}

	// Test 5: Ensure -w 0 is NOT used (prevents race conditions with concurrent window creation)
	if strings.Contains(argsStr, `-w 0`) {
		t.Errorf("Expected '-w 0' to be removed to avoid race conditions, got: %s", argsStr)
	}

	// Test 6: Ensure --suppressApplicationTitle is NOT used (can cause silent failures on older WT)
	if strings.Contains(argsStr, `--suppressApplicationTitle`) {
		t.Errorf("Expected '--suppressApplicationTitle' to be removed, got: %s", argsStr)
	}
}
