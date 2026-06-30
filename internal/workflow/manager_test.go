package workflow

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tanmayythakare/creo/internal/config"
	"github.com/tanmayythakare/creo/internal/ui"
)

func TestFindPorts(t *testing.T) {
	tests := []struct {
		cmd      string
		expected []int
	}{
		{"npm run dev --port 3000", []int{3000}},
		{"python app.py -p 8080 -d 9090", []int{8080, 9090}},
		{"go run main.go", nil},
		{"mvn spring-boot:run -Dspring-boot.run.arguments=--server.port=8082", []int{8082}},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			got := findPorts(tt.cmd)
			if len(got) != len(tt.expected) {
				t.Fatalf("expected %v, got %v", tt.expected, got)
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("at index %d: expected %d, got %d", i, tt.expected[i], got[i])
				}
			}
		})
	}
}

func TestIsPortInUse(t *testing.T) {
	// Find a free port by listening and then closing
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find a free port: %v", err)
	}
	addr := listener.Addr().(*net.TCPAddr)
	port := addr.Port

	// Test 1: Port is not in use (since we closed it or will close it now)
	listener.Close()
	if isPortInUse(port) {
		t.Errorf("expected port %d to be free, but it was reported as in use", port)
	}

	// Test 2: Port is in use
	listener, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("failed to bind port %d: %v", port, err)
	}
	defer listener.Close()

	if !isPortInUse(port) {
		t.Errorf("expected port %d to be in use, but it was reported as free", port)
	}
}

func TestResolveCommand(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("resolve maven wrapper mvnw.cmd", func(t *testing.T) {
		// Temporarily clear PATH so global binaries like mvn are not resolved via LookPath
		oldPath := os.Getenv("PATH")
		os.Setenv("PATH", "")
		defer os.Setenv("PATH", oldPath)

		svcPath := "java-app"
		appPath := filepath.Join(tempDir, svcPath)
		_ = os.MkdirAll(appPath, 0755)

		// Create mock mvnw.cmd in service directory
		wrapperFile := filepath.Join(appPath, "mvnw.cmd")
		_ = os.WriteFile(wrapperFile, []byte(""), 0644)

		svc := config.Service{
			Path:    svcPath,
			Command: "mvn spring-boot:run",
		}

		ui.SetReader(strings.NewReader("\n")) // bypass any UI logs warnings
		resolved := ResolveCommand("api", svc, appPath, tempDir)

		expectedCmd := filepath.Join(".", "mvnw.cmd") + " spring-boot:run"
		if resolved.Command != expectedCmd {
			t.Errorf("expected command %q, got %q", expectedCmd, resolved.Command)
		}
	})

	t.Run("resolve node local binary", func(t *testing.T) {
		// Temporarily clear PATH so global binaries like vite are not resolved via LookPath
		oldPath := os.Getenv("PATH")
		os.Setenv("PATH", "")
		defer os.Setenv("PATH", oldPath)

		svcPath := "node-app"
		appPath := filepath.Join(tempDir, svcPath)
		nodeBinPath := filepath.Join(appPath, "node_modules", ".bin")
		_ = os.MkdirAll(nodeBinPath, 0755)

		// Create mock vite.cmd inside local node_modules/.bin
		viteFile := filepath.Join(nodeBinPath, "vite.cmd")
		_ = os.WriteFile(viteFile, []byte(""), 0644)

		svc := config.Service{
			Path:    svcPath,
			Command: "vite dev",
		}

		resolved := ResolveCommand("web", svc, appPath, tempDir)

		expectedCmd := filepath.Join(".", "node_modules", ".bin", "vite.cmd") + " dev"
		if resolved.Command != expectedCmd {
			t.Errorf("expected command %q, got %q", expectedCmd, resolved.Command)
		}
	})

	t.Run("no wrapper command exists in PATH", func(t *testing.T) {
		// go is standard, should be found in system PATH (or won't be replaced if no local exists)
		svc := config.Service{
			Path:    ".",
			Command: "go run main.go",
		}
		resolved := ResolveCommand("api", svc, tempDir, tempDir)
		if resolved.Command != "go run main.go" {
			t.Errorf("expected command unchanged, got: %s", resolved.Command)
		}
	})
}

func TestStart(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("dry run workflow", func(t *testing.T) {
		cfg := &config.Config{
			Services: map[string]config.Service{
				"web": {
					Path:    ".",
					Command: "npm run dev",
				},
			},
			Workflows: map[string][]string{
				"dev": {"web"},
			},
		}

		err := Start(cfg, "dev", []string{"web"}, tempDir, true, false)
		if err != nil {
			t.Fatalf("unexpected dry-run error: %v", err)
		}
	})

	t.Run("nil/missing services workflow", func(t *testing.T) {
		cfg := &config.Config{
			Services: map[string]config.Service{},
		}

		err := Start(cfg, "dev", []string{"web"}, tempDir, false, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
