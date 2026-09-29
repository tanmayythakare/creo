package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("valid creo.yml", func(t *testing.T) {
		content := `
project:
  name: test-proj
services:
  app:
    command: run.sh
workflows:
  default:
    - app
`
		os.WriteFile(filepath.Join(tempDir, "creo.yml"), []byte(content), 0644)
		cfg, err := Load(tempDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Project.Name != "test-proj" {
			t.Errorf("expected test-proj, got %s", cfg.Project.Name)
		}
	})

	t.Run("missing project.name", func(t *testing.T) {
		content := `
project:
services:
  app:
    command: run.sh
`
		os.WriteFile(filepath.Join(tempDir, "creo.yml"), []byte(content), 0644)
		_, err := Load(tempDir)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("no services", func(t *testing.T) {
		content := `
project:
  name: test
services:
`
		os.WriteFile(filepath.Join(tempDir, "creo.yml"), []byte(content), 0644)
		_, err := Load(tempDir)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("workflow references unknown service", func(t *testing.T) {
		content := `
project:
  name: test
services:
  app:
    command: run.sh
workflows:
  default:
    - unknown
`
		os.WriteFile(filepath.Join(tempDir, "creo.yml"), []byte(content), 0644)
		_, err := Load(tempDir)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("duplicate keys in yaml", func(t *testing.T) {
		content := `
project:
  name: test
services:
  app:
    command: run.sh
  app:
    command: other.sh
`
		os.WriteFile(filepath.Join(tempDir, "creo.yml"), []byte(content), 0644)
		_, err := Load(tempDir)
		if err == nil {
			t.Fatal("expected error for duplicate key, got nil")
		}
	})

	t.Run("command injection rejection", func(t *testing.T) {
		content := `
project:
  name: test
services:
  app:
    command: run.sh && malicious.sh
`
		os.WriteFile(filepath.Join(tempDir, "creo.yml"), []byte(content), 0644)
		_, err := Load(tempDir)
		if err == nil {
			t.Fatal("expected error for command injection, got nil")
		}
	})

	t.Run("path traversal rejection in service name", func(t *testing.T) {
		content := `
project:
  name: test
services:
  ../../app:
    command: run.sh
`
		os.WriteFile(filepath.Join(tempDir, "creo.yml"), []byte(content), 0644)
		_, err := Load(tempDir)
		if err == nil {
			t.Fatal("expected error for path traversal in service name, got nil")
		}
	})

	t.Run("workflow as string rejection", func(t *testing.T) {
		content := `
project:
  name: test
services:
  app:
    command: run.sh
workflows:
  default: app
`
		os.WriteFile(filepath.Join(tempDir, "creo.yml"), []byte(content), 0644)
		_, err := Load(tempDir)
		if err == nil {
			t.Fatal("expected error for workflow as string, got nil")
		}
	})

	t.Run("invalid service name rejection", func(t *testing.T) {
		content := `
project:
  name: test
services:
  app'service:
    command: run.sh
`
		os.WriteFile(filepath.Join(tempDir, "creo.yml"), []byte(content), 0644)
		_, err := Load(tempDir)
		if err == nil {
			t.Fatal("expected error for invalid characters in service name, got nil")
		}
	})

	t.Run("YAML reserved word rejection", func(t *testing.T) {
		content := `
project:
  name: test
services:
  no:
    command: run.sh
`
		os.WriteFile(filepath.Join(tempDir, "creo.yml"), []byte(content), 0644)
		_, err := Load(tempDir)
		if err == nil {
			t.Fatal("expected error for YAML reserved word, got nil")
		}
	})

	t.Run("single ampersand command rejection", func(t *testing.T) {
		content := `
project:
  name: test
services:
  app:
    command: run.sh & malicious.sh
`
		os.WriteFile(filepath.Join(tempDir, "creo.yml"), []byte(content), 0644)
		_, err := Load(tempDir)
		if err == nil {
			t.Fatal("expected error for command containing single ampersand, got nil")
		}
	})

	t.Run("newline command rejection", func(t *testing.T) {
		content := `
project:
  name: test
services:
  app:
    command: "run.sh\nmalicious.sh"
`
		os.WriteFile(filepath.Join(tempDir, "creo.yml"), []byte(content), 0644)
		_, err := Load(tempDir)
		if err == nil {
			t.Fatal("expected error for command containing newline, got nil")
		}
	})

	t.Run("workflow name conflicts with drop builtin", func(t *testing.T) {
		content := `
project:
  name: test
services:
  app:
    command: run.sh
workflows:
  drop:
    - app
`
		os.WriteFile(filepath.Join(tempDir, "creo.yml"), []byte(content), 0644)
		_, err := Load(tempDir)
		if err == nil {
			t.Fatal("expected error for workflow name conflicting with 'drop', got nil")
		}
	})

	t.Run("workflow name conflicts with delete builtin", func(t *testing.T) {
		content := `
project:
  name: test
services:
  app:
    command: run.sh
workflows:
  delete:
    - app
`
		os.WriteFile(filepath.Join(tempDir, "creo.yml"), []byte(content), 0644)
		_, err := Load(tempDir)
		if err == nil {
			t.Fatal("expected error for workflow name conflicting with 'delete', got nil")
		}
	})
}

