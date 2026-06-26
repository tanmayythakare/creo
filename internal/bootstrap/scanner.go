package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
)

// maxScanDepth is how many directory levels below root we scan.
// Depth 1 = backend/, depth 2 = frontend/secondbrain-frontend/, depth 3 = packages/apps/web/.
const maxScanDepth = 3

// maxDirsScanned is a safety valve — if a repo has more directories than this,
// we stop scanning and proceed with whatever was found so far. Silent, no error.
const maxDirsScanned = 500

// ScanResult holds all discovered indicator files per directory
type ScanResult struct {
	Root    string
	Entries []DirEntry
}

// DirEntry represents a scanned subdirectory
type DirEntry struct {
	Path  string // relative path from root
	Files []string
}

// knownIndicators are files that tell us something about a service
var knownIndicators = []string{
	"package.json",
	"pom.xml",
	"mvnw",
	"mvnw.cmd",
	"build.gradle",
	"build.gradle.kts",
	"go.mod",
	"requirements.txt",
	"Pipfile",
	"pyproject.toml",
	"Cargo.toml",
	"composer.json",
	"docker-compose.yml",
	"docker-compose.yaml",
	"Dockerfile",
	".env",
	"angular.json",
	"next.config.js",
	"next.config.ts",
	"nuxt.config.js",
	"nuxt.config.ts",
	"vite.config.js",
	"vite.config.ts",
	"svelte.config.js",
	"remix.config.js",
	"gatsby-config.js",
	"astro.config.js",
	"astro.config.mjs",
	"vue.config.js",
	"quasar.config.js",
	"nest-cli.json",
	"manage.py",
	"artisan",
	"Gemfile",
	"pubspec.yaml",
	"turbo.json",
	"nx.json",
	"lerna.json",
	"Makefile",
	"Procfile",
	"deno.json",
	"deno.jsonc",
	"bun.lockb",
	"webpack.config.js",
	"*.csproj",
	"*.sln",
	"Program.cs",
	"index.html",
	"app.py",
	"main.py",
	"main.go",
	"index.php",
	"*.java",
	"app.rb",
	"Podfile",
}

// shouldSkipDir returns true for directories that should never be scanned at any depth.
func shouldSkipDir(name string) bool {
	if len(name) == 0 {
		return true
	}
	if name[0] == '.' {
		return true
	}
	switch name {
	case "node_modules", "vendor", "target", "dist", "build":
		return true
	}
	return false
}

// Scan walks up to maxScanDepth levels deep from root and collects indicator files.
// Root is special: even if it contains indicator files, we still recurse into subdirectories
// (monorepos often have a root package.json for workspace config plus services in subdirs).
// Non-root directories prune on indicator found — if a dir has indicator files, we don't
// descend further (that directory IS the service).
func Scan(root string) (*ScanResult, error) {
	result := &ScanResult{Root: root}
	scanned := 0

	// Check root itself
	rootEntry, err := scanDir(root, ".")
	if err == nil && len(rootEntry.Files) > 0 {
		result.Entries = append(result.Entries, rootEntry)
	}
	scanned++

	// Recurse into subdirectories
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if !e.IsDir() || shouldSkipDir(e.Name()) {
			continue
		}
		scanRecursive(result, root, e.Name(), 1, &scanned)
	}

	return result, nil
}

// scanRecursive walks a directory tree, collecting indicator files.
// depth tracks current level (1 = immediate child of root).
// scanned is a shared counter for the hard cap guardrail.
func scanRecursive(result *ScanResult, root, relPath string, depth int, scanned *int) {
	// Guardrail 3: hard cap on total directories scanned
	if *scanned >= maxDirsScanned {
		return
	}
	*scanned++

	absPath := filepath.Join(root, relPath)

	// Scan this directory for indicator files
	entry, err := scanDir(absPath, relPath)
	if err != nil {
		return
	}

	// Guardrail 2: if indicators found, add entry and DON'T recurse deeper.
	// This directory is a service — no need to scan its src/ or lib/ subdirs.
	if len(entry.Files) > 0 {
		result.Entries = append(result.Entries, entry)
		return
	}

	// No indicators here — recurse into subdirectories if depth allows
	if depth >= maxScanDepth {
		return
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// Guardrail 1: skip-list applies at EVERY depth
		if shouldSkipDir(e.Name()) {
			continue
		}
		childRel := filepath.Join(relPath, e.Name())
		scanRecursive(result, root, childRel, depth+1, scanned)
	}
}

func scanDir(absPath, relPath string) (DirEntry, error) {
	entry := DirEntry{Path: relPath}

	for _, indicator := range knownIndicators {
		if strings.Contains(indicator, "*") {
			files, err := os.ReadDir(absPath)
			if err == nil {
				for _, f := range files {
					if !f.IsDir() {
						matched, errMatch := filepath.Match(indicator, f.Name())
						if errMatch == nil && matched {
							entry.Files = append(entry.Files, f.Name())
						}
					}
				}
			}
		} else {
			full := filepath.Join(absPath, indicator)
			if _, err := os.Stat(full); err == nil {
				entry.Files = append(entry.Files, indicator)
			}
		}
	}

	return entry, nil
}
