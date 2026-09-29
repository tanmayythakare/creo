package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tanmayythakare/creo/internal/ui"
)

func TestDropAliasDelete(t *testing.T) {
	hasDeleteAlias := false
	for _, alias := range dropCmd.Aliases {
		if alias == "delete" {
			hasDeleteAlias = true
			break
		}
	}
	if !hasDeleteAlias {
		t.Errorf("expected dropCmd to have 'delete' alias, got %v", dropCmd.Aliases)
	}
}

func TestDropNoCreoFiles(t *testing.T) {
	tempDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to chdir to tempDir: %v", err)
	}

	err = runDrop(dropCmd, []string{})
	if err != nil {
		t.Fatalf("expected nil error when no creo files exist, got: %v", err)
	}
}

func TestDropInteractiveAbort(t *testing.T) {
	tempDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to chdir to tempDir: %v", err)
	}

	configPath := filepath.Join(tempDir, "creo.yml")
	if err := os.WriteFile(configPath, []byte("project:\n  name: test\n"), 0644); err != nil {
		t.Fatalf("failed to write creo.yml: %v", err)
	}

	creoDir := filepath.Join(tempDir, ".creo")
	if err := os.MkdirAll(creoDir, 0755); err != nil {
		t.Fatalf("failed to create .creo dir: %v", err)
	}

	forceDrop = false
	dryRun = false
	ui.SetReader(strings.NewReader("n\n"))

	err = runDrop(dropCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error on abort: %v", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("expected creo.yml to remain intact after abort, but it was removed")
	}
	if _, err := os.Stat(creoDir); os.IsNotExist(err) {
		t.Errorf("expected .creo/ to remain intact after abort, but it was removed")
	}
}

func TestDropForce(t *testing.T) {
	tempDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(origDir)
		forceDrop = false
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to chdir to tempDir: %v", err)
	}

	configPath := filepath.Join(tempDir, "creo.yml")
	if err := os.WriteFile(configPath, []byte("project:\n  name: test\n"), 0644); err != nil {
		t.Fatalf("failed to write creo.yml: %v", err)
	}

	creoDir := filepath.Join(tempDir, ".creo")
	if err := os.MkdirAll(creoDir, 0755); err != nil {
		t.Fatalf("failed to create .creo dir: %v", err)
	}

	forceDrop = true
	dryRun = false

	err = runDrop(dropCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error on force drop: %v", err)
	}

	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Errorf("expected creo.yml to be removed, but it still exists")
	}
	if _, err := os.Stat(creoDir); !os.IsNotExist(err) {
		t.Errorf("expected .creo/ to be removed, but it still exists")
	}
}

func TestDropDryRun(t *testing.T) {
	tempDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(origDir)
		dryRun = false
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to chdir to tempDir: %v", err)
	}

	configPath := filepath.Join(tempDir, "creo.yml")
	if err := os.WriteFile(configPath, []byte("project:\n  name: test\n"), 0644); err != nil {
		t.Fatalf("failed to write creo.yml: %v", err)
	}

	creoDir := filepath.Join(tempDir, ".creo")
	if err := os.MkdirAll(creoDir, 0755); err != nil {
		t.Fatalf("failed to create .creo dir: %v", err)
	}

	dryRun = true
	forceDrop = false

	err = runDrop(dropCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error on dry run: %v", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("expected creo.yml to remain intact during dry-run, but it was deleted")
	}
	if _, err := os.Stat(creoDir); os.IsNotExist(err) {
		t.Errorf("expected .creo/ to remain intact during dry-run, but it was deleted")
	}
}
