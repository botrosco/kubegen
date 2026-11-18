package generator_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kubegen/pkg/generator"
	"kubegen/pkg/parser"
	"kubegen/pkg/tpl"
)

// render runs a bundled template the way the CLI does: load it, parse its
// frontmatter, and generate into a directory.
func render(t *testing.T, templateName string, args []string) string {
	t.Helper()

	content, err := tpl.GetContent(templateName)
	if err != nil {
		t.Fatalf("loading template %s: %v", templateName, err)
	}
	defs, body, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("parsing template %s: %v", templateName, err)
	}

	out := t.TempDir()
	full := append([]string{}, args...)
	full = append(full, "-o", out)
	if err := generator.GenerateManifestsWith(defs, body, generator.Options{
		Args:   full,
		Stdin:  strings.NewReader(""),
		Stdout: &strings.Builder{},
	}); err != nil {
		t.Fatalf("generating %s: %v", templateName, err)
	}
	return out
}

// compareGolden asserts that the render produced exactly the files in the golden
// directory, with identical content.
func compareGolden(t *testing.T, gotDir, goldenDir string) {
	t.Helper()

	want, err := os.ReadDir(goldenDir)
	if err != nil {
		t.Fatalf("reading golden dir: %v", err)
	}

	seen := map[string]bool{}
	for _, entry := range want {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		seen[name] = true

		wantBytes, err := os.ReadFile(filepath.Join(goldenDir, name))
		if err != nil {
			t.Fatalf("reading golden %s: %v", name, err)
		}
		gotBytes, err := os.ReadFile(filepath.Join(gotDir, name))
		if err != nil {
			t.Errorf("expected file %s was not generated: %v", name, err)
			continue
		}
		if string(gotBytes) != string(wantBytes) {
			t.Errorf("file %s differs from golden.\n--- want ---\n%s\n--- got ---\n%s", name, wantBytes, gotBytes)
		}
	}

	entries, err := os.ReadDir(gotDir)
	if err != nil {
		t.Fatalf("reading output dir: %v", err)
	}
	for _, entry := range entries {
		if !seen[entry.Name()] {
			t.Errorf("unexpected file generated: %s", entry.Name())
		}
	}
}

// TestAppBundleGolden pins the whole bundle: deployment, service, routes, secret,
// image tracking and the folder kustomization.
func TestAppBundleGolden(t *testing.T) {
	out := render(t, "app", []string{"-f", filepath.Join("testdata", "app-values.yaml")})
	compareGolden(t, out, filepath.Join("testdata", "app.golden"))
}

// TestNamespaceGolden pins the namespace scaffolding: the Namespace plus the
// kustomization carrying the image-automation patch.
func TestNamespaceGolden(t *testing.T) {
	out := render(t, "namespace", []string{"-Name=golden", "-Interval=50m"})
	compareGolden(t, out, filepath.Join("testdata", "namespace.golden"))
}

// TestAppBundleKustomizationMatchesFiles guards the invariant the consuming repo
// depends on: the emitted kustomization lists exactly the files that were written.
func TestAppBundleKustomizationMatchesFiles(t *testing.T) {
	out := render(t, "app", []string{"-f", filepath.Join("testdata", "app-values.yaml")})

	kustomization, err := os.ReadFile(filepath.Join(out, "kustomization.yaml"))
	if err != nil {
		t.Fatalf("reading kustomization: %v", err)
	}

	list := map[string]bool{}
	for _, line := range strings.Split(string(kustomization), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") && strings.HasSuffix(trimmed, ".yaml") {
			name := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			list[name] = true
			if _, err := os.Stat(filepath.Join(out, name)); err != nil {
				t.Errorf("kustomization lists %s, which was not written: %v", name, err)
			}
		}
	}

	entries, _ := os.ReadDir(out)
	for _, entry := range entries {
		name := entry.Name()
		if name == "kustomization.yaml" {
			continue
		}
		if !list[name] {
			t.Errorf("file %s is not listed in the kustomization", name)
		}
	}
	if strings.Contains(string(kustomization), "namespace:") {
		t.Errorf("the app kustomization must not set namespace (the parent does):\n%s", kustomization)
	}
}

// TestNamespaceRequiresInterval checks that the colliding-default hazard is
// impossible: the interval has no default, so a caller must choose one.
func TestNamespaceRequiresInterval(t *testing.T) {
	content, err := tpl.GetContent("namespace")
	if err != nil {
		t.Fatalf("loading template: %v", err)
	}
	defs, body, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("parsing template: %v", err)
	}
	err = generator.GenerateManifestsWith(defs, body, generator.Options{
		Args:   []string{"-Name=golden"},
		Stdin:  strings.NewReader(""),
		Stdout: &strings.Builder{},
	})
	if err == nil || !strings.Contains(err.Error(), "Interval") {
		t.Fatalf("expected a missing-Interval error, got %v", err)
	}
}
