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

	t.Run("nested subdirectory depth 2", func(t *testing.T) {
		rootDir := filepath.Join(tempDir, "nested2")
		nestedDir := filepath.Join(rootDir, "frontend", "my-app")
		os.MkdirAll(nestedDir, 0755)
		os.WriteFile(filepath.Join(nestedDir, "package.json"), []byte("{}"), 0644)

		res, err := Scan(rootDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res.Entries) != 1 {
			t.Errorf("expected 1 entry, got %d", len(res.Entries))
			return
		}
		wantPath := filepath.Join("frontend", "my-app")
		if res.Entries[0].Path != wantPath {
			t.Errorf("expected path %q, got %q", wantPath, res.Entries[0].Path)
		}
	})

	t.Run("deeply nested depth 3", func(t *testing.T) {
		rootDir := filepath.Join(tempDir, "nested3")
		nestedDir := filepath.Join(rootDir, "packages", "apps", "web")
		os.MkdirAll(nestedDir, 0755)
		os.WriteFile(filepath.Join(nestedDir, "package.json"), []byte("{}"), 0644)

		res, err := Scan(rootDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res.Entries) != 1 {
			t.Errorf("expected 1 entry, got %d", len(res.Entries))
			return
		}
		wantPath := filepath.Join("packages", "apps", "web")
		if res.Entries[0].Path != wantPath {
			t.Errorf("expected path %q, got %q", wantPath, res.Entries[0].Path)
		}
	})

	t.Run("beyond max depth 4 not found", func(t *testing.T) {
		rootDir := filepath.Join(tempDir, "nested4")
		nestedDir := filepath.Join(rootDir, "a", "b", "c", "d")
		os.MkdirAll(nestedDir, 0755)
		os.WriteFile(filepath.Join(nestedDir, "package.json"), []byte("{}"), 0644)

		res, err := Scan(rootDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res.Entries) != 0 {
			t.Errorf("expected 0 entries (beyond max depth), got %d", len(res.Entries))
		}
	})

	t.Run("prune on indicator found", func(t *testing.T) {
		rootDir := filepath.Join(tempDir, "prune")
		backend := filepath.Join(rootDir, "backend")
		backendSrc := filepath.Join(backend, "src")
		os.MkdirAll(backendSrc, 0755)
		os.WriteFile(filepath.Join(backend, "pom.xml"), []byte("<project/>"), 0644)
		os.WriteFile(filepath.Join(backendSrc, "main.py"), []byte(""), 0644)

		res, err := Scan(rootDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should find only backend (with pom.xml), NOT backend/src (with main.py)
		if len(res.Entries) != 1 {
			t.Errorf("expected 1 entry (pruned), got %d", len(res.Entries))
			for _, e := range res.Entries {
				t.Logf("  entry: %s files=%v", e.Path, e.Files)
			}
			return
		}
		if res.Entries[0].Path != "backend" {
			t.Errorf("expected path 'backend', got %q", res.Entries[0].Path)
		}
	})

	t.Run("skip-list applies at depth 2", func(t *testing.T) {
		rootDir := filepath.Join(tempDir, "skipdeep")
		nestedSkip := filepath.Join(rootDir, "apps", "node_modules")
		os.MkdirAll(nestedSkip, 0755)
		os.WriteFile(filepath.Join(nestedSkip, "package.json"), []byte("{}"), 0644)

		res, err := Scan(rootDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res.Entries) != 0 {
			t.Errorf("expected 0 entries (node_modules at depth 2 should be skipped), got %d", len(res.Entries))
		}
	})
}

