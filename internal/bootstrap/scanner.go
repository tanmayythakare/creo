package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
)

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

// Scan walks one level deep from root and collects indicator files
func Scan(root string) (*ScanResult, error) {
	result := &ScanResult{Root: root}

	// Check root itself
	rootEntry, err := scanDir(root, ".")
	if err == nil && len(rootEntry.Files) > 0 {
		result.Entries = append(result.Entries, rootEntry)
	}

	// Check immediate subdirectories
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		// skip hidden dirs and common non-service dirs
		if name[0] == '.' || name == "node_modules" || name == "vendor" || name == "target" || name == "dist" || name == "build" {
			continue
		}
		dirPath := filepath.Join(root, name)
		entry, err := scanDir(dirPath, name)
		if err != nil {
			continue
		}
		if len(entry.Files) > 0 {
			result.Entries = append(result.Entries, entry)
		}
	}

	return result, nil
}

func scanDir(absPath, relPath string) (DirEntry, error) {
	entry := DirEntry{Path: relPath}

	for _, indicator := range knownIndicators {
		if strings.Contains(indicator, "*") {
			matches, err := filepath.Glob(filepath.Join(absPath, indicator))
			if err == nil && len(matches) > 0 {
				for _, m := range matches {
					entry.Files = append(entry.Files, filepath.Base(m))
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
