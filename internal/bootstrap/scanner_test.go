package bootstrap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScan(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("empty directory", func(t *testing.T) {
		emptyDir := filepath.Join(tempDir, "empty")
		os.Mkdir(emptyDir, 0755)
		res, err := Scan(emptyDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res.Entries) != 0 {
			t.Errorf("expected 0 entries, got %d", len(res.Entries))
		}
	})

	t.Run("directory with package.json", func(t *testing.T) {
		pkgDir := filepath.Join(tempDir, "pkg")
		os.Mkdir(pkgDir, 0755)
		os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte("{}"), 0644)
		res, err := Scan(pkgDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res.Entries) != 1 {
			t.Errorf("expected 1 entry, got %d", len(res.Entries))
		}
		if res.Entries[0].Files[0] != "package.json" {
			t.Errorf("expected package.json, got %s", res.Entries[0].Files[0])
		}
	})

	t.Run("directory with pom.xml", func(t *testing.T) {
		pomDir := filepath.Join(tempDir, "pom")
		os.Mkdir(pomDir, 0755)
		os.WriteFile(filepath.Join(pomDir, "pom.xml"), []byte("<project/>"), 0644)
		res, err := Scan(pomDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res.Entries) != 1 {
			t.Errorf("expected 1 entry, got %d", len(res.Entries))
		}
		if res.Entries[0].Files[0] != "pom.xml" {
			t.Errorf("expected pom.xml, got %s", res.Entries[0].Files[0])
		}
	})

	t.Run("skips node_modules vendor dist", func(t *testing.T) {
		skipDir := filepath.Join(tempDir, "skip")
		os.Mkdir(skipDir, 0755)
		
		for _, name := range []string{"node_modules", "vendor", "dist"} {
			d := filepath.Join(skipDir, name)
			os.Mkdir(d, 0755)
			os.WriteFile(filepath.Join(d, "package.json"), []byte("{}"), 0644)
		}
		
		res, err := Scan(skipDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res.Entries) != 0 {
			t.Errorf("expected 0 entries, got %d", len(res.Entries))
		}
	})
}
