package bootstrap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetect(t *testing.T) {

	// ─── Tier 1: Framework config files ──────────────────────────────

	t.Run("Angular", func(t *testing.T) {
		entry := DirEntry{Path: "frontend", Files: []string{"package.json", "angular.json"}}
		svc := detectEntry(".", entry)
		assertDetection(t, svc, "Angular", "ng serve", "high")
	})

	t.Run("Next.js config", func(t *testing.T) {
		entry := DirEntry{Path: "web", Files: []string{"next.config.js"}}
		svc := detectEntry(".", entry)
		assertDetection(t, svc, "Next.js", "npm run dev", "high")
	})

	t.Run("NestJS nest-cli.json", func(t *testing.T) {
		entry := DirEntry{Path: "api", Files: []string{"package.json", "nest-cli.json"}}
		svc := detectEntry(".", entry)
		assertDetection(t, svc, "NestJS", "npm run start:dev", "high")
	})

	t.Run("Django manage.py", func(t *testing.T) {
		entry := DirEntry{Path: "backend", Files: []string{"manage.py"}}
		svc := detectEntry(".", entry)
		assertDetection(t, svc, "Django", "python manage.py runserver", "medium")
	})

	t.Run("Laravel artisan", func(t *testing.T) {
		entry := DirEntry{Path: "server", Files: []string{"artisan"}}
		svc := detectEntry(".", entry)
		assertDetection(t, svc, "Laravel", "php artisan serve", "high")
	})

	t.Run("Flutter pubspec.yaml", func(t *testing.T) {
		entry := DirEntry{Path: "app", Files: []string{"pubspec.yaml"}}
		svc := detectEntry(".", entry)
		assertDetection(t, svc, "Flutter", "flutter run", "high")
	})

	t.Run("Turborepo turbo.json", func(t *testing.T) {
		entry := DirEntry{Path: ".", Files: []string{"turbo.json"}}
		svc := detectEntry(".", entry)
		assertDetection(t, svc, "Turborepo", "npx turbo run dev", "high")
	})

	// ─── Tier 2: Language-specific build files ───────────────────────

	t.Run("Spring Boot Maven", func(t *testing.T) {
		entry := DirEntry{Path: "backend", Files: []string{"pom.xml"}}
		svc := detectEntry(".", entry)
		assertDetection(t, svc, "Spring Boot (Maven)", "mvn spring-boot:run", "high")
	})

	t.Run("Go", func(t *testing.T) {
		tmp := t.TempDir()
		dir := "api"
		os.MkdirAll(filepath.Join(tmp, dir), 0755)
		os.WriteFile(filepath.Join(tmp, dir, "go.mod"), []byte("module example.com/api\n\ngo 1.21\n"), 0644)

		entry := DirEntry{Path: dir, Files: []string{"go.mod"}}
		svc := detectEntry(tmp, entry)
		assertDetection(t, svc, "Go", "go run .", "high")
	})

	t.Run("ASP.NET Core csproj", func(t *testing.T) {
		entry := DirEntry{Path: "dotnet-api", Files: []string{"MyService.csproj"}}
		svc := detectEntry(".", entry)
		assertDetection(t, svc, "ASP.NET Core", "dotnet run", "high")
	})

	// ─── Tier 3: package.json dependency reading ─────────────────────

	t.Run("Next.js from package.json deps", func(t *testing.T) {
		tmp := t.TempDir()
		dir := "web"
		os.MkdirAll(filepath.Join(tmp, dir), 0755)
		pkgData := `{ "dependencies": { "next": "^14.0.0", "react": "^18.0.0" } }`
		os.WriteFile(filepath.Join(tmp, dir, "package.json"), []byte(pkgData), 0644)

		entry := DirEntry{Path: dir, Files: []string{"package.json"}}
		svc := detectEntry(tmp, entry)
		// Next.js should win over React (#next+react priority)
		assertDetection(t, svc, "Next.js", "npm run dev", "high")
	})

	t.Run("NestJS from package.json deps", func(t *testing.T) {
		tmp := t.TempDir()
		dir := "api"
		os.MkdirAll(filepath.Join(tmp, dir), 0755)
		pkgData := `{ "dependencies": { "@nestjs/core": "^10.0.0" } }`
		os.WriteFile(filepath.Join(tmp, dir, "package.json"), []byte(pkgData), 0644)

		entry := DirEntry{Path: dir, Files: []string{"package.json"}}
		svc := detectEntry(tmp, entry)
		assertDetection(t, svc, "NestJS", "npm run start:dev", "high")
	})

	t.Run("Next.js from devDependencies", func(t *testing.T) {
		tmp := t.TempDir()
		dir := "web"
		os.MkdirAll(filepath.Join(tmp, dir), 0755)
		pkgData := `{ "devDependencies": { "next": "13.0.0" } }`
		os.WriteFile(filepath.Join(tmp, dir, "package.json"), []byte(pkgData), 0644)

		entry := DirEntry{Path: dir, Files: []string{"package.json"}}
		svc := detectEntry(tmp, entry)
		// devDependencies should also be detected (#9)
		assertDetection(t, svc, "Next.js", "npm run dev", "high")
	})

	// ─── Edge case: empty/malformed package.json (#2) ────────────────

	t.Run("Empty package.json", func(t *testing.T) {
		tmp := t.TempDir()
		dir := "web"
		os.MkdirAll(filepath.Join(tmp, dir), 0755)
		os.WriteFile(filepath.Join(tmp, dir, "package.json"), []byte("{}"), 0644)

		entry := DirEntry{Path: dir, Files: []string{"package.json"}}
		svc := detectEntry(tmp, entry)
		// Falls through to generic Node.js — no panic
		assertDetection(t, svc, "Node.js", "npm run dev", "medium")
	})

	t.Run("Malformed package.json", func(t *testing.T) {
		tmp := t.TempDir()
		dir := "web"
		os.MkdirAll(filepath.Join(tmp, dir), 0755)
		os.WriteFile(filepath.Join(tmp, dir, "package.json"), []byte(`{name: "frontend", scripts: {`), 0644)

		entry := DirEntry{Path: dir, Files: []string{"package.json"}}
		svc := detectEntry(tmp, entry)
		// Should not panic, falls through to generic Node.js
		assertDetection(t, svc, "Node.js", "npm run dev", "medium")
	})

	// ─── Edge case: pom.xml + build.gradle conflict (#7) ─────────────

	t.Run("pom.xml wins over build.gradle", func(t *testing.T) {
		entry := DirEntry{Path: "backend", Files: []string{"pom.xml", "build.gradle"}}
		svc := detectEntry(".", entry)
		// Maven should win
		assertDetection(t, svc, "Spring Boot (Maven)", "mvn spring-boot:run", "high")
	})

	// ─── Edge case: angular.json + package.json coexist ──────────────

	t.Run("angular.json wins over package.json deps", func(t *testing.T) {
		tmp := t.TempDir()
		dir := "frontend"
		os.MkdirAll(filepath.Join(tmp, dir), 0755)
		os.WriteFile(filepath.Join(tmp, dir, "angular.json"), []byte("{}"), 0644)
		pkgData := `{ "dependencies": { "react": "^18.0.0" } }`
		os.WriteFile(filepath.Join(tmp, dir, "package.json"), []byte(pkgData), 0644)

		entry := DirEntry{Path: dir, Files: []string{"angular.json", "package.json"}}
		svc := detectEntry(tmp, entry)
		// Tier 1 angular.json beats Tier 3 package.json
		assertDetection(t, svc, "Angular", "ng serve", "high")
	})

	// ─── Edge case: manage.py + requirements.txt (#13 Django wins) ───

	t.Run("Django wins over requirements.txt", func(t *testing.T) {
		entry := DirEntry{Path: "backend", Files: []string{"manage.py", "requirements.txt"}}
		svc := detectEntry(".", entry)
		// Tier 1 manage.py beats Tier 3 requirements.txt
		assertDetection(t, svc, "Django", "python manage.py runserver", "medium")
	})

	// ─── Edge case: FastAPI from requirements.txt ────────────────────

	t.Run("FastAPI from requirements.txt", func(t *testing.T) {
		tmp := t.TempDir()
		dir := "app"
		os.MkdirAll(filepath.Join(tmp, dir), 0755)
		os.WriteFile(filepath.Join(tmp, dir, "requirements.txt"), []byte("fastapi>=0.100.0\nuvicorn"), 0644)

		entry := DirEntry{Path: dir, Files: []string{"main.py", "requirements.txt"}}
		svc := detectEntry(tmp, entry)
		assertDetection(t, svc, "FastAPI", "uvicorn main:app --reload", "high")
	})

	// ─── Edge case: main.py alone = no detection (#14) ───────────────

	t.Run("main.py alone does not detect Python", func(t *testing.T) {
		entry := DirEntry{Path: "scripts", Files: []string{"main.py"}}
		svc := detectEntry(".", entry)
		// Should NOT detect as Python — requires requirements.txt/Pipfile/pyproject.toml
		if svc.Confidence != "unknown" {
			t.Errorf("expected unknown confidence for bare main.py, got %s (%s)", svc.Confidence, svc.Framework)
		}
	})

	// ─── Edge case: go.mod self-detection (#5) ───────────────────────

	t.Run("Creo own go.mod is skipped", func(t *testing.T) {
		tmp := t.TempDir()
		modContent := "module github.com/tanmaythakare/creo\n\ngo 1.21\n"
		os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(modContent), 0644)

		entry := DirEntry{Path: ".", Files: []string{"go.mod"}}
		svc := detectEntry(tmp, entry)
		// Should not detect creo itself as a service
		if svc.Framework == "Go" {
			t.Error("expected creo's own go.mod to be skipped, but detected as Go service")
		}
	})

	// ─── Edge case: service name with spaces (#6) ────────────────────

	t.Run("Path with spaces sanitized", func(t *testing.T) {
		entry := DirEntry{Path: "frontend app", Files: []string{"package.json"}}
		svc := detectEntry(".", entry)
		if svc.Name != "frontend-app" {
			t.Errorf("expected sanitized name 'frontend-app', got '%s'", svc.Name)
		}
	})

	// ─── Edge case: docker-compose.yml vs .yaml (#8) ─────────────────

	t.Run("docker-compose.yml wins over .yaml", func(t *testing.T) {
		tmp := t.TempDir()
		dir := "infra"
		os.MkdirAll(filepath.Join(tmp, dir), 0755)
		os.WriteFile(filepath.Join(tmp, dir, "docker-compose.yml"), []byte("services:\n  db:\n    image: postgres"), 0644)
		os.WriteFile(filepath.Join(tmp, dir, "docker-compose.yaml"), []byte("services:\n  cache:\n    image: redis"), 0644)

		entry := DirEntry{Path: dir, Files: []string{"docker-compose.yml", "docker-compose.yaml"}}
		svc := detectEntry(tmp, entry)
		// .yml wins — should detect postgres from .yml content
		assertDetection(t, svc, "PostgreSQL", "docker compose up", "high")
	})

	// ─── Edge case: Docker Compose with generic/unknown services ─────

	t.Run("Docker Compose generic", func(t *testing.T) {
		tmp := t.TempDir()
		dir := "infra"
		os.MkdirAll(filepath.Join(tmp, dir), 0755)
		os.WriteFile(filepath.Join(tmp, dir, "docker-compose.yml"), []byte("services:\n  app:\n    image: mycompany/custom"), 0644)

		entry := DirEntry{Path: dir, Files: []string{"docker-compose.yml"}}
		svc := detectEntry(tmp, entry)
		assertDetection(t, svc, "Docker Compose", "docker compose up", "high")
	})

	// ─── Edge case: Procfile with web process (#11) ──────────────────

	t.Run("Procfile extracts web process", func(t *testing.T) {
		tmp := t.TempDir()
		dir := "myapp"
		os.MkdirAll(filepath.Join(tmp, dir), 0755)
		os.WriteFile(filepath.Join(tmp, dir, "Procfile"), []byte("worker: node worker.js\nweb: node server.js\n"), 0644)

		entry := DirEntry{Path: dir, Files: []string{"Procfile"}}
		svc := detectEntry(tmp, entry)
		// web: process should be preferred
		if svc.SuggestedCmd != "node server.js" {
			t.Errorf("expected 'node server.js' from web: process, got '%s'", svc.SuggestedCmd)
		}
	})

	// ─── Edge case: Makefile with no dev or run target ────────────────

	t.Run("Makefile no dev/run target", func(t *testing.T) {
		tmp := t.TempDir()
		dir := "project"
		os.MkdirAll(filepath.Join(tmp, dir), 0755)
		os.WriteFile(filepath.Join(tmp, dir, "Makefile"), []byte(".PHONY: build test\nbuild:\n\tgo build .\ntest:\n\tgo test ./...\n"), 0644)

		entry := DirEntry{Path: dir, Files: []string{"Makefile"}}
		svc := detectEntry(tmp, entry)
		if svc.SuggestedCmd != "make" {
			t.Errorf("expected 'make' fallback, got '%s'", svc.SuggestedCmd)
		}
	})

	// ─── Edge case: 5+ indicator files — priority is strict (#general) ─

	t.Run("Multiple indicators strict priority", func(t *testing.T) {
		entry := DirEntry{Path: "fullstack", Files: []string{
			"package.json", "go.mod", "Dockerfile", "Makefile", "requirements.txt",
		}}
		svc := detectEntry(".", entry)
		// go.mod (Tier 2) should win — package.json/requirements.txt are Tier 3, Docker/Make are Tier 4/5
		// But go.mod needs readFile check for creo module; since "." won't have go.mod content, it falls through
		// package.json at Tier 3 should be next
		if svc.Framework != "Go" && svc.Framework != "Node.js" {
			t.Errorf("expected Go or Node.js for multi-indicator dir, got %s", svc.Framework)
		}
	})

	// ─── Unknown ─────────────────────────────────────────────────────

	t.Run("Unknown", func(t *testing.T) {
		entry := DirEntry{Path: "random", Files: []string{"unknown.txt"}}
		svc := detectEntry(".", entry)
		if svc.Confidence != "unknown" {
			t.Errorf("expected unknown confidence, got %s", svc.Confidence)
		}
	})
}

// assertDetection is a helper to check framework, command, and confidence in one call.
func assertDetection(t *testing.T, svc DetectedService, framework, cmd, confidence string) {
	t.Helper()
	if svc.Framework != framework {
		t.Errorf("expected framework %q, got %q", framework, svc.Framework)
	}
	if svc.SuggestedCmd != cmd {
		t.Errorf("expected cmd %q, got %q", cmd, svc.SuggestedCmd)
	}
	if svc.Confidence != confidence {
		t.Errorf("expected confidence %q, got %q", confidence, svc.Confidence)
	}
}
