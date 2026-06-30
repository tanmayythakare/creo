package config

import (
	"path/filepath"
	"testing"
)

func TestWrite(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "creo.yml")

	t.Run("write valid config", func(t *testing.T) {
		cfg := &Config{
			Project: Project{Name: "test-project"},
			Services: map[string]Service{
				"web": {
					Command: "npm start",
				},
			},
			Workflows: map[string][]string{
				"default": {"web"},
			},
		}

		err := Write(cfg, configPath)
		if err != nil {
			t.Fatalf("unexpected error writing config: %v", err)
		}

		// Reload to verify it was written correctly
		loaded, err := Load(tempDir)
		if err != nil {
			t.Fatalf("failed to load written config: %v", err)
		}

		if loaded.Project.Name != "test-project" {
			t.Errorf("expected test-project, got %s", loaded.Project.Name)
		}
		if svc, ok := loaded.Services["web"]; !ok || svc.Command != "npm start" {
			t.Errorf("service 'web' not written correctly: %+v", svc)
		}
	})

	t.Run("write to invalid path", func(t *testing.T) {
		cfg := &Config{
			Project: Project{Name: "error-project"},
		}

		// Non-existent directory path
		invalidPath := filepath.Join(tempDir, "non_existent_dir", "creo.yml")
		err := Write(cfg, invalidPath)
		if err == nil {
			t.Fatal("expected error writing to invalid path, got nil")
		}
	})

	t.Run("overwrite existing config", func(t *testing.T) {
		cfg1 := &Config{
			Project: Project{Name: "project-v1"},
			Services: map[string]Service{
				"api": {Command: "go run main.go"},
			},
		}
		cfg2 := &Config{
			Project: Project{Name: "project-v2"},
			Services: map[string]Service{
				"api": {Command: "go run main.go"},
			},
		}

		err := Write(cfg1, configPath)
		if err != nil {
			t.Fatalf("failed to write initial config: %v", err)
		}

		err = Write(cfg2, configPath)
		if err != nil {
			t.Fatalf("failed to overwrite config: %v", err)
		}

		loaded, err := Load(tempDir)
		if err != nil {
			t.Fatalf("failed to load overwritten config: %v", err)
		}

		if loaded.Project.Name != "project-v2" {
			t.Errorf("expected project-v2 after overwrite, got %s", loaded.Project.Name)
		}
	})
}
