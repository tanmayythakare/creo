package bootstrap

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// maxFileReadSize is the maximum file size (1 MB) we'll read for framework detection.
// Prevents memory issues with massive monorepo package.json files. (#3)
const maxFileReadSize = 1 * 1024 * 1024

// creoModuleName is used to skip self-detection when creo's own go.mod is found at the project root. (#5)
const creoModuleName = "github.com/tanmaythakare/creo"

// DetectedService is what the detector infers about a directory
type DetectedService struct {
	Path         string
	Name         string
	Framework    string
	SuggestedCmd string
	Confidence   string // "high", "medium", "unknown"
	Alias        string
}

// Detect runs over scan results and produces DetectedService for each entry
func Detect(scan *ScanResult) []DetectedService {
	var services []DetectedService

	for _, entry := range scan.Entries {
		svc := detectEntry(scan.Root, entry)
		services = append(services, svc)
	}

	return services
}

// readFile reads a file from root/dir/filename with a 1 MB size cap (#3).
// Returns empty string on any error — malformed content is handled gracefully
// downstream via strings.Contains, never via JSON/YAML unmarshal (#2, #4).
func readFile(root, dir, filename string) string {
	abs := filepath.Join(root, dir, filename)
	fi, err := os.Stat(abs)
	if err != nil {
		return ""
	}
	// Cap at maxFileReadSize to avoid memory issues with massive files (#3)
	if fi.Size() > maxFileReadSize {
		return ""
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return ""
	}
	return string(data)
}

// sanitizeServiceName replaces spaces and invalid characters with hyphens
// so the name is safe for YAML keys and wt.exe commands (#6).
func sanitizeServiceName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '-', r == '_', r == '.':
			b.WriteRune(r)
		case r == ' ', r == '\t':
			b.WriteRune('-')
		default:
			// drop other special characters
		}
	}
	result := b.String()
	if result == "" {
		return "service"
	}
	return result
}

func detectEntry(root string, entry DirEntry) DetectedService {
	fileSet := toSet(entry.Files)

	svc := DetectedService{
		Path:       entry.Path,
		Name:       sanitizeServiceName(entry.Path), // (#6) sanitize spaces/special chars
		Confidence: "unknown",
	}

	// ─── Priority Tier 1: Framework-specific config files ───────────────
	// These are unambiguous — if the config file exists, we know the framework.
	// Tier 1 always beats Tier 2/3/4/5 when multiple indicators coexist (#7, Angular+package.json, Django+requirements.txt).

	if fileSet["angular.json"] {
		svc.Framework = "Angular"
		svc.SuggestedCmd = "ng serve"
		svc.Confidence = "high"
		return svc
	}
	if fileSet["next.config.js"] || fileSet["next.config.ts"] {
		svc.Framework = "Next.js"
		svc.SuggestedCmd = "npm run dev"
		svc.Confidence = "high"
		return svc
	}
	if fileSet["nuxt.config.js"] || fileSet["nuxt.config.ts"] {
		svc.Framework = "Nuxt.js"
		svc.SuggestedCmd = "npm run dev"
		svc.Confidence = "high"
		return svc
	}
	if fileSet["nest-cli.json"] {
		svc.Framework = "NestJS"
		svc.SuggestedCmd = "npm run start:dev"
		svc.Confidence = "high"
		return svc
	}
	// manage.py → likely Django but could be a custom script (#13).
	// We set "medium" confidence and note this in the wizard.
	if fileSet["manage.py"] {
		svc.Framework = "Django"
		svc.SuggestedCmd = "python manage.py runserver"
		svc.Confidence = "medium"
		return svc
	}
	if fileSet["artisan"] {
		svc.Framework = "Laravel"
		svc.SuggestedCmd = "php artisan serve"
		svc.Confidence = "high"
		return svc
	}
	if fileSet["svelte.config.js"] {
		svc.Framework = "SvelteKit"
		svc.SuggestedCmd = "npm run dev"
		svc.Confidence = "high"
		return svc
	}
	if fileSet["remix.config.js"] {
		svc.Framework = "Remix"
		svc.SuggestedCmd = "npm run dev"
		svc.Confidence = "high"
		return svc
	}
	if fileSet["gatsby-config.js"] {
		svc.Framework = "Gatsby"
		svc.SuggestedCmd = "gatsby develop"
		svc.Confidence = "high"
		return svc
	}
	if fileSet["astro.config.js"] || fileSet["astro.config.mjs"] {
		svc.Framework = "Astro"
		svc.SuggestedCmd = "npm run dev"
		svc.Confidence = "high"
		return svc
	}
	if fileSet["vue.config.js"] {
		svc.Framework = "Vue CLI"
		svc.SuggestedCmd = "npm run serve"
		svc.Confidence = "high"
		return svc
	}
	if fileSet["quasar.config.js"] {
		svc.Framework = "Quasar"
		svc.SuggestedCmd = "quasar dev"
		svc.Confidence = "high"
		return svc
	}
	if fileSet["webpack.config.js"] {
		svc.Framework = "Webpack"
		svc.SuggestedCmd = "npm run dev"
		svc.Confidence = "high"
		return svc
	}
	if fileSet["pubspec.yaml"] {
		svc.Framework = "Flutter"
		svc.SuggestedCmd = "flutter run"
		svc.Confidence = "high"
		return svc
	}
	if fileSet["Podfile"] {
		svc.Framework = "iOS/React Native"
		svc.SuggestedCmd = "npx react-native run-ios"
		svc.Confidence = "high"
		return svc
	}
	if fileSet["turbo.json"] {
		svc.Framework = "Turborepo"
		svc.SuggestedCmd = "npx turbo run dev"
		svc.Confidence = "high"
		return svc
	}
	if fileSet["nx.json"] {
		svc.Framework = "Nx Monorepo"
		svc.SuggestedCmd = "npx nx serve"
		svc.Confidence = "high"
		return svc
	}
	if fileSet["lerna.json"] {
		svc.Framework = "Lerna"
		svc.SuggestedCmd = "npm run dev"
		svc.Confidence = "high"
		return svc
	}

	// ─── Priority Tier 2: Language-specific build files ─────────────────
	// pom.xml is checked before build.gradle so Maven wins the conflict (#7).
	if fileSet["pom.xml"] {
		svc.Framework = "Spring Boot (Maven)"
		svc.SuggestedCmd = "mvn spring-boot:run"
		svc.Confidence = "high"
		return svc
	}
	if fileSet["build.gradle"] || fileSet["build.gradle.kts"] {
		svc.Framework = "Spring Boot (Gradle)"
		svc.SuggestedCmd = "./gradlew bootRun"
		svc.Confidence = "high"
		return svc
	}
	if fileSet["go.mod"] {
		// Skip self-detection: if go.mod at root contains creo's own module name (#5)
		modContent := readFile(root, entry.Path, "go.mod")
		if isCreoModule(modContent) {
			// Don't detect creo itself — fall through to next tier
		} else {
			framework, cmd := checkGoFramework(root, entry.Path, fileSet)
			svc.Framework = framework
			svc.SuggestedCmd = cmd
			svc.Confidence = "high"
			return svc
		}
	}
	if fileSet["Cargo.toml"] {
		svc.Framework = "Rust/Actix"
		svc.SuggestedCmd = "cargo run"
		svc.Confidence = "high"
		return svc
	}
	// .csproj glob may return multiple files — we use the first match (#10).
	// dotnet run auto-picks the project if only one exists; warns if multiple.
	if hasCsproj(entry.Files) {
		svc.Framework = "ASP.NET Core"
		svc.SuggestedCmd = "dotnet run"
		svc.Confidence = "high"
		return svc
	}
	if hasSln(entry.Files) {
		svc.Framework = ".NET Solution"
		svc.SuggestedCmd = "dotnet run"
		svc.Confidence = "high"
		return svc
	}
	if fileSet["Program.cs"] {
		svc.Framework = ".NET"
		svc.SuggestedCmd = "dotnet run"
		svc.Confidence = "high"
		return svc
	}

	// ─── Priority Tier 3: Generic package managers ──────────────────────
	// package.json: we do text search via strings.Contains — NOT json.Unmarshal.
	// This means malformed JSON won't panic (#2) and devDependencies are
	// automatically included because we search the raw file content (#9).
	if fileSet["package.json"] {
		content := readFile(root, entry.Path, "package.json")
		if detectPackageJSON(content, &svc) {
			return svc
		}
		// Fallback: generic Node.js
		svc.Framework = "Node.js"
		svc.SuggestedCmd = "npm run dev"
		svc.Confidence = "medium"
		return svc
	}
	if fileSet["requirements.txt"] || fileSet["Pipfile"] || fileSet["pyproject.toml"] {
		framework, cmd := checkPythonFramework(root, entry.Path, fileSet)
		if framework != "" {
			svc.Framework = framework
			svc.SuggestedCmd = cmd
			svc.Confidence = "high"
		} else {
			svc.Framework = "Python"
			svc.SuggestedCmd = "python main.py"
			svc.Confidence = "medium"
		}
		return svc
	}
	if fileSet["Gemfile"] {
		framework, cmd := checkRubyFramework(root, entry.Path)
		svc.Framework = framework
		svc.SuggestedCmd = cmd
		svc.Confidence = "medium"
		return svc
	}
	if fileSet["composer.json"] {
		svc.Framework = "PHP/Composer"
		svc.SuggestedCmd = "php -S localhost:8000"
		svc.Confidence = "medium"
		return svc
	}

	// ─── Priority Tier 4: Docker/Container files ────────────────────────
	// docker-compose.yml is checked before .yaml — .yml wins the conflict (#8).
	// Content is inspected via strings.Contains — malformed YAML won't panic (#4).
	if fileSet["docker-compose.yml"] || fileSet["docker-compose.yaml"] {
		filename := "docker-compose.yml"
		if !fileSet["docker-compose.yml"] {
			filename = "docker-compose.yaml"
		}
		content := readFile(root, entry.Path, filename)
		detectDockerCompose(content, &svc)
		return svc
	}
	if fileSet["Dockerfile"] {
		svc.Framework = "Docker"
		svc.SuggestedCmd = "docker compose up"
		svc.Confidence = "medium"
		return svc
	}

	// ─── Priority Tier 5: Fallbacks ─────────────────────────────────────
	if fileSet["deno.json"] || fileSet["deno.jsonc"] {
		svc.Framework = "Deno"
		svc.SuggestedCmd = "deno run main.ts"
		svc.Confidence = "medium"
		return svc
	}
	if fileSet["bun.lockb"] {
		svc.Framework = "Bun"
		svc.SuggestedCmd = "bun run dev"
		svc.Confidence = "medium"
		return svc
	}
	if fileSet["Makefile"] {
		svc.Framework = "Make"
		svc.SuggestedCmd = checkMakefileTarget(root, entry.Path)
		svc.Confidence = "medium"
		return svc
	}
	// Procfile: parse processes and use first web: process if available (#11).
	if fileSet["Procfile"] {
		svc.Framework = "Heroku/Foreman"
		svc.SuggestedCmd = parseProcfileCommand(root, entry.Path)
		svc.Confidence = "medium"
		return svc
	}

	// ─── Priority Tier 6: Bare language files ───────────────────────────
	// index.html only counts if no framework config was found above (prevents false positive inside dist/)
	if fileSet["index.html"] {
		svc.Framework = "HTML/CSS/JS"
		svc.SuggestedCmd = "npx serve ."
		svc.Confidence = "medium"
		return svc
	}
	if fileSet["index.php"] {
		svc.Framework = "PHP"
		svc.SuggestedCmd = "php -S localhost:8000"
		svc.Confidence = "medium"
		return svc
	}
	// Bare .java files (no pom.xml or build.gradle) — standalone Java
	if hasJava(entry.Files) {
		mainClass := findJavaMainFile(entry.Files)
		svc.Framework = "Java"
		svc.SuggestedCmd = "java " + mainClass
		svc.Confidence = "medium"
		return svc
	}
	// main.py and app.py alone (no requirements.txt/Pipfile/pyproject.toml)
	// are NOT detected as Python services to avoid false positives (#14).
	// They only contribute to detection when paired with a package manager file in Tier 3.

	return svc
}

// ──────────────────────────────────────────────────────────────────────────────
// Helper functions
// ──────────────────────────────────────────────────────────────────────────────

func toSet(files []string) map[string]bool {
	set := make(map[string]bool)
	for _, f := range files {
		set[f] = true
	}
	return set
}

func hasCsproj(files []string) bool {
	for _, f := range files {
		if strings.HasSuffix(f, ".csproj") {
			return true
		}
	}
	return false
}

func hasSln(files []string) bool {
	for _, f := range files {
		if strings.HasSuffix(f, ".sln") {
			return true
		}
	}
	return false
}

func hasJava(files []string) bool {
	for _, f := range files {
		if strings.HasSuffix(f, ".java") {
			return true
		}
	}
	return false
}

func findJavaMainFile(files []string) string {
	for _, f := range files {
		if strings.HasSuffix(f, ".java") {
			return strings.TrimSuffix(f, ".java")
		}
	}
	return "Main"
}

// isCreoModule returns true if the go.mod content declares creo's own module (#5).
func isCreoModule(modContent string) bool {
	for _, line := range strings.Split(modContent, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.Contains(line, creoModuleName)
		}
	}
	return false
}

// detectPackageJSON searches raw file content for known dependency strings.
// Uses strings.Contains — NOT json.Unmarshal — so malformed JSON is safe (#2).
// This also covers devDependencies automatically since we search the entire file (#9).
func detectPackageJSON(content string, svc *DetectedService) bool {
	if content == "" {
		return false
	}

	if strings.Contains(content, `"@angular/core"`) {
		svc.Framework = "Angular"
		svc.SuggestedCmd = "ng serve"
		svc.Confidence = "high"
		return true
	}
	if strings.Contains(content, `"next"`) {
		svc.Framework = "Next.js"
		svc.SuggestedCmd = "npm run dev"
		svc.Confidence = "high"
		return true
	}
	if strings.Contains(content, `"nuxt"`) {
		svc.Framework = "Nuxt.js"
		svc.SuggestedCmd = "npm run dev"
		svc.Confidence = "high"
		return true
	}
	if strings.Contains(content, `"svelte"`) || strings.Contains(content, `"@sveltejs/kit"`) {
		svc.Framework = "SvelteKit"
		svc.SuggestedCmd = "npm run dev"
		svc.Confidence = "high"
		return true
	}
	if strings.Contains(content, `"@nestjs/core"`) {
		svc.Framework = "NestJS"
		svc.SuggestedCmd = "npm run start:dev"
		svc.Confidence = "high"
		return true
	}
	if strings.Contains(content, `"express"`) {
		svc.Framework = "Express.js"
		svc.SuggestedCmd = "npm run dev"
		svc.Confidence = "medium"
		return true
	}
	if strings.Contains(content, `"fastify"`) {
		svc.Framework = "Fastify"
		svc.SuggestedCmd = "npm run dev"
		svc.Confidence = "medium"
		return true
	}
	if strings.Contains(content, `"@hapi/hapi"`) {
		svc.Framework = "Hapi.js"
		svc.SuggestedCmd = "npm run dev"
		svc.Confidence = "medium"
		return true
	}
	if strings.Contains(content, `"gatsby"`) {
		svc.Framework = "Gatsby"
		svc.SuggestedCmd = "gatsby develop"
		svc.Confidence = "high"
		return true
	}
	// React without Next/Gatsby — check last so Next.js wins the conflict (#next vs react priority)
	if strings.Contains(content, `"react"`) && !strings.Contains(content, `"next"`) && !strings.Contains(content, `"gatsby"`) {
		svc.Framework = "React (Vite or CRA)"
		svc.SuggestedCmd = "npm run dev"
		svc.Confidence = "medium"
		return true
	}
	if strings.Contains(content, `"vue"`) && !strings.Contains(content, `"nuxt"`) {
		svc.Framework = "Vue"
		svc.SuggestedCmd = "npm run dev"
		svc.Confidence = "medium"
		return true
	}
	return false
}

func checkGoFramework(root, dir string, fileSet map[string]bool) (string, string) {
	contentMod := readFile(root, dir, "go.mod")
	contentMain := ""
	if fileSet["main.go"] {
		contentMain = readFile(root, dir, "main.go")
	}

	content := strings.ToLower(contentMod + " " + contentMain)
	if strings.Contains(content, "github.com/gin-gonic/gin") || strings.Contains(content, `"gin"`) {
		return "Gin", "go run ."
	}
	if strings.Contains(content, "github.com/gofiber/fiber") || strings.Contains(content, `"fiber"`) {
		return "Fiber", "go run ."
	}
	return "Go", "go run ."
}

func checkPythonFramework(root, dir string, fileSet map[string]bool) (string, string) {
	content := ""
	if fileSet["requirements.txt"] {
		content = readFile(root, dir, "requirements.txt")
	} else if fileSet["Pipfile"] {
		content = readFile(root, dir, "Pipfile")
	} else if fileSet["pyproject.toml"] {
		content = readFile(root, dir, "pyproject.toml")
	}

	contentLower := strings.ToLower(content)
	if strings.Contains(contentLower, "fastapi") {
		return "FastAPI", "uvicorn main:app --reload"
	}
	if strings.Contains(contentLower, "flask") {
		return "Flask", "flask run"
	}
	return "", ""
}

func checkRubyFramework(root, dir string) (string, string) {
	content := strings.ToLower(readFile(root, dir, "Gemfile"))
	if strings.Contains(content, "rails") {
		return "Rails", "rails server"
	}
	if strings.Contains(content, "sinatra") {
		return "Sinatra", "ruby app.rb"
	}
	return "Ruby", "ruby main.rb"
}

func checkMakefileTarget(root, dir string) string {
	content := readFile(root, dir, "Makefile")
	if strings.Contains(content, "dev:") {
		return "make dev"
	}
	if strings.Contains(content, "run:") {
		return "make run"
	}
	// No dev or run target — suggest make with a note
	return "make"
}

// parseProcfileCommand reads the Procfile and extracts the web: process command.
// If no web: process, uses the first process found (#11).
func parseProcfileCommand(root, dir string) string {
	content := readFile(root, dir, "Procfile")
	if content == "" {
		return "foreman start"
	}

	var firstCmd string
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		processType := strings.TrimSpace(parts[0])
		cmd := strings.TrimSpace(parts[1])
		// Prefer the web: process
		if processType == "web" {
			return cmd
		}
		if firstCmd == "" {
			firstCmd = cmd
		}
	}

	if firstCmd != "" {
		return firstCmd
	}
	return "foreman start"
}

// detectDockerCompose inspects raw content for known database image names.
// Uses strings.Contains — NOT yaml.Unmarshal — so malformed YAML is safe (#4).
func detectDockerCompose(content string, svc *DetectedService) {
	svc.SuggestedCmd = "docker compose up"
	svc.Confidence = "high"
	contentLower := strings.ToLower(content)
	if strings.Contains(contentLower, "postgres") || strings.Contains(contentLower, "postgresql") {
		svc.Framework = "PostgreSQL"
	} else if strings.Contains(contentLower, "mongo") || strings.Contains(contentLower, "mongodb") {
		svc.Framework = "MongoDB"
	} else if strings.Contains(contentLower, "mysql") {
		svc.Framework = "MySQL"
	} else if strings.Contains(contentLower, "redis") {
		svc.Framework = "Redis"
	} else if strings.Contains(contentLower, "mariadb") {
		svc.Framework = "MariaDB"
	} else {
		svc.Framework = "Docker Compose"
	}
}
