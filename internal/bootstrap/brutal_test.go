package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// brutalTest describes one real-world scenario for end-to-end Scan+Detect testing.
type brutalTest struct {
	name   string
	setup  func(dir string) error // creates files in the temp dir
	want   []string               // expected Framework strings in results (at least one)
	noWant []string               // frameworks that must NOT appear
	wantN  int                    // minimum number of detected services (0 = don't check)
}

var brutalTests = []brutalTest{
	{
		name: "Test1_NextJS_Only",
		setup: func(dir string) error {
			return os.WriteFile(
				filepath.Join(dir, "package.json"),
				[]byte(`{"dependencies":{"next":"13.0.0","react":"18.0.0"}}`),
				0644,
			)
		},
		want: []string{"Next.js"},
	},
	{
		name: "Test2_NestJS_Backend_Only",
		setup: func(dir string) error {
			if err := os.WriteFile(filepath.Join(dir, "nest-cli.json"), []byte(`{}`), 0644); err != nil {
				return err
			}
			return os.WriteFile(
				filepath.Join(dir, "package.json"),
				[]byte(`{"dependencies":{"@nestjs/core":"10.0.0"}}`),
				0644,
			)
		},
		want: []string{"NestJS"},
	},
	{
		name: "Test3_Django_Project",
		setup: func(dir string) error {
			if err := os.WriteFile(filepath.Join(dir, "manage.py"), []byte(""), 0644); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("django>=4.0\n"), 0644)
		},
		want: []string{"Django"},
	},
	{
		name: "Test4_Fullstack_NextJS_SpringBoot_Docker",
		setup: func(dir string) error {
			fe := filepath.Join(dir, "frontend")
			be := filepath.Join(dir, "backend")
			db := filepath.Join(dir, "database")
			for _, d := range []string{fe, be, db} {
				if err := os.MkdirAll(d, 0755); err != nil {
					return err
				}
			}
			if err := os.WriteFile(filepath.Join(fe, "package.json"), []byte(`{"dependencies":{"next":"13.0.0"}}`), 0644); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(be, "pom.xml"), []byte(`<project></project>`), 0644); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(db, "docker-compose.yml"), []byte("services:\n  postgres:\n    image: postgres\n"), 0644)
		},
		want:  []string{"Next.js", "Spring Boot (Maven)", "PostgreSQL"},
		wantN: 3,
	},
	{
		name: "Test5_Monorepo_Turbo",
		setup: func(dir string) error {
			if err := os.WriteFile(filepath.Join(dir, "turbo.json"), []byte(`{"pipeline":{}}`), 0644); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies":{"next":"13.0.0"}}`), 0644)
		},
		// turbo.json is Tier 1 and wins at root level over package.json
		want: []string{"Turborepo"},
	},
	{
		name: "Test6_Laravel_Plus_Vue",
		setup: func(dir string) error {
			if err := os.WriteFile(filepath.Join(dir, "artisan"), []byte(""), 0644); err != nil {
				return err
			}
			return os.WriteFile(
				filepath.Join(dir, "package.json"),
				[]byte(`{"dependencies":{"vue":"3.0.0"}}`),
				0644,
			)
		},
		// artisan is Tier 1 -> Laravel wins
		want: []string{"Laravel"},
	},
	{
		name: "Test7_Flutter_Mobile",
		setup: func(dir string) error {
			return os.WriteFile(filepath.Join(dir, "pubspec.yaml"), []byte("name: myapp\n"), 0644)
		},
		want: []string{"Flutter"},
	},
	{
		name: "Test8_DotNet_Project",
		setup: func(dir string) error {
			return os.WriteFile(filepath.Join(dir, "MyApp.csproj"), []byte(""), 0644)
		},
		want: []string{"ASP.NET Core"},
	},
	{
		name: "Test9_Malformed_PackageJSON",
		setup: func(dir string) error {
			return os.WriteFile(filepath.Join(dir, "package.json"), []byte("{broken json here"), 0644)
		},
		// Must not panic; falls back to generic Node.js
		want: []string{"Node.js"},
	},
	{
		name: "Test10_Empty_Directory",
		setup: func(dir string) error {
			return nil // nothing to create
		},
		// No services should be detected in a truly empty directory
		wantN:  0,
		noWant: []string{"Next.js", "Django", "Go", "Laravel", "Flutter"},
	},
	{
		name: "Test11_Nested_Frontend_Backend",
		setup: func(dir string) error {
			be := filepath.Join(dir, "backend")
			fe := filepath.Join(dir, "frontend", "secondbrain-frontend")
			for _, d := range []string{be, fe} {
				if err := os.MkdirAll(d, 0755); err != nil {
					return err
				}
			}
			if err := os.WriteFile(filepath.Join(be, "pom.xml"),
				[]byte(`<project></project>`), 0644); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(fe, "angular.json"),
				[]byte(`{}`), 0644); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(fe, "package.json"),
				[]byte(`{"dependencies":{"@angular/core":"16.0.0"}}`), 0644)
		},
		want:  []string{"Spring Boot (Maven)", "Angular"},
		wantN: 2,
	},
}

// TestBrutalScenarios runs Scan + Detect end-to-end on real temp directories,
// simulating the 10 brutal real-world project types.
func TestBrutalScenarios(t *testing.T) {
	for _, tc := range brutalTests {
		tc := tc // capture range var
		t.Run(tc.name, func(t *testing.T) {
			// Create a fresh temp directory for this test
			dir := t.TempDir()
			if err := tc.setup(dir); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			// Run the real Scan + Detect pipeline
			scan, err := Scan(dir)
			if err != nil {
				t.Fatalf("Scan() error: %v", err)
			}
			detected := Detect(scan)

			// Collect detected frameworks for assertions
			got := make([]string, 0, len(detected))
			for _, svc := range detected {
				got = append(got, svc.Framework)
			}
			gotStr := strings.Join(got, ", ")
			if gotStr == "" {
				gotStr = "(none)"
			}
			t.Logf("detected: [%s]", gotStr)

			// Check minimum service count
			if tc.wantN > 0 && len(detected) < tc.wantN {
				t.Errorf("expected at least %d services, got %d: [%s]", tc.wantN, len(detected), gotStr)
			}
			// Empty test: expect zero services
			if tc.wantN == 0 && len(tc.want) == 0 && len(detected) != 0 {
				t.Errorf("expected 0 services for empty dir, got %d: [%s]", len(detected), gotStr)
			}

			// Check that every wanted framework appears in results
			for _, w := range tc.want {
				found := false
				for _, svc := range detected {
					if svc.Framework == w {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected framework %q in results, got: [%s]", w, gotStr)
				}
			}

			// Check that forbidden frameworks do NOT appear
			for _, bad := range tc.noWant {
				for _, svc := range detected {
					if svc.Framework == bad {
						t.Errorf("unexpected framework %q in results: [%s]", bad, gotStr)
					}
				}
			}
		})
	}
}
