package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kubegen/pkg/parser"
)

const simpleTemplate = `values:
  - name: Name
    type: string
    required: true
  - name: Image
    type: string
    default: "nginx:latest"
  - name: Port
    type: int
    default: 80
  - name: Hostnames
    type: list
  - name: EmitNamespace
    type: bool
    default: false
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Name }}
{{- if .EmitNamespace }}
  namespace: apps
{{- end }}
spec:
  template:
    spec:
      containers:
        - name: {{ .Name }}
          image: {{ .Image }}
          ports:
            - containerPort: {{ .Port }}
          {{- if .Hostnames }}
          env:
          {{- range .Hostnames }}
            - name: {{ . }}
              value: x
          {{- end }}
          {{- end }}
`

func generate(t *testing.T, tmplContent string, args []string, stdin string) error {
	t.Helper()
	defs, body, err := parser.Parse(tmplContent)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return GenerateManifestsWith(defs, body, Options{
		Args:   args,
		Stdin:  strings.NewReader(stdin),
		Stdout: &strings.Builder{},
	})
}

func writeValues(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "values.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing values file: %v", err)
	}
	return path
}

func TestPrecedence(t *testing.T) {
	values := writeValues(t, "Image: from-file:1.0\n")
	dir := t.TempDir()

	// values file beats the default
	if err := generate(t, simpleTemplate, []string{"-Name=demo", "-f", values, "-o", dir}, ""); err != nil {
		t.Fatalf("generate: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "demo-deployment.yaml"))
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	if !strings.Contains(string(got), "image: from-file:1.0") {
		t.Errorf("values file should override the default; got:\n%s", got)
	}

	// an explicit flag beats the values file
	if err := generate(t, simpleTemplate, []string{"-Name=demo", "-f", values, "-Image=from-flag:2.0", "-o", dir}, ""); err != nil {
		t.Fatalf("generate: %v", err)
	}
	got, _ = os.ReadFile(filepath.Join(dir, "demo-deployment.yaml"))
	if !strings.Contains(string(got), "image: from-flag:2.0") {
		t.Errorf("explicit flag should override the values file; got:\n%s", got)
	}
}

func TestValuesFromStdin(t *testing.T) {
	dir := t.TempDir()
	if err := generate(t, simpleTemplate, []string{"-Name=demo", "-f", "-", "-o", dir}, "Image: stdin:3.0\n"); err != nil {
		t.Fatalf("generate: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "demo-deployment.yaml"))
	if !strings.Contains(string(got), "image: stdin:3.0") {
		t.Errorf("stdin values were not applied; got:\n%s", got)
	}
}

func TestUnknownValuesKey(t *testing.T) {
	values := writeValues(t, "Imgae: typo:1.0\n")
	err := generate(t, simpleTemplate, []string{"-Name=demo", "-f", values}, "")
	if err == nil || !strings.Contains(err.Error(), "Imgae") {
		t.Fatalf("expected an unknown-value error naming the key, got %v", err)
	}
}

func TestMissingValuesFile(t *testing.T) {
	err := generate(t, simpleTemplate, []string{"-Name=demo", "-f", "/nonexistent/values.yaml"}, "")
	if err == nil || !strings.Contains(err.Error(), "values file") {
		t.Fatalf("expected a read error, got %v", err)
	}
}

func TestRequiredAfterMerge(t *testing.T) {
	// required value satisfied by the values file alone
	values := writeValues(t, "Name: from-file\n")
	if err := generate(t, simpleTemplate, []string{"-f", values}, ""); err != nil {
		t.Fatalf("a required value supplied by the values file should satisfy the check: %v", err)
	}
	// missing everywhere
	err := generate(t, simpleTemplate, nil, "")
	if err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("expected a required-value error, got %v", err)
	}
}

func TestListValueHasNoFlag(t *testing.T) {
	err := generate(t, simpleTemplate, []string{"-Name=demo", "-Hostnames=a,b"}, "")
	if err == nil || !strings.Contains(err.Error(), "Hostnames") {
		t.Fatalf("expected an undefined-flag error for a list value, got %v", err)
	}
}

func TestNonScalarFromValuesFile(t *testing.T) {
	values := writeValues(t, "Hostnames: [a.rossd.net, b.rossd.net]\n")
	dir := t.TempDir()
	if err := generate(t, simpleTemplate, []string{"-Name=demo", "-f", values, "-o", dir}, ""); err != nil {
		t.Fatalf("generate: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "demo-deployment.yaml"))
	for _, want := range []string{"name: a.rossd.net", "name: b.rossd.net"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("expected %q in output:\n%s", want, got)
		}
	}
}

func TestUniformFileNaming(t *testing.T) {
	tmpl := `values:
  - name: Name
    type: string
    required: true
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Name }}
---
apiVersion: v1
kind: Service
metadata:
  name: {{ .Name }}
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Name }}-cm
---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
metadata:
  name: {{ .Name }}
  annotations:
    kubegen.io/filename: kustomization.yaml
`
	dir := t.TempDir()
	if err := generate(t, tmpl, []string{"-Name=app", "-o", dir}, ""); err != nil {
		t.Fatalf("generate: %v", err)
	}
	entries, _ := os.ReadDir(dir)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	want := []string{"app-cm-configmap.yaml", "app-deployment.yaml", "app-service.yaml", "kustomization.yaml"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("file names = %v, want %v", names, want)
	}

	// the filename annotation must not leak into the written file
	body, _ := os.ReadFile(filepath.Join(dir, "kustomization.yaml"))
	if strings.Contains(string(body), "kubegen.io/filename") || strings.Contains(string(body), "annotations:") {
		t.Errorf("kubegen.io/filename leaked into the output:\n%s", body)
	}
}

func TestEmitNamespaceDefault(t *testing.T) {
	dir := t.TempDir()
	if err := generate(t, simpleTemplate, []string{"-Name=demo", "-o", dir}, ""); err != nil {
		t.Fatalf("generate: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "demo-deployment.yaml"))
	if strings.Contains(string(got), "namespace:") {
		t.Errorf("metadata.namespace should not be emitted by default:\n%s", got)
	}

	if err := generate(t, simpleTemplate, []string{"-Name=demo", "-EmitNamespace=true", "-o", dir}, ""); err != nil {
		t.Fatalf("generate: %v", err)
	}
	got, _ = os.ReadFile(filepath.Join(dir, "demo-deployment.yaml"))
	if !strings.Contains(string(got), "namespace: apps") {
		t.Errorf("EmitNamespace=true should emit the namespace:\n%s", got)
	}
}

func TestSplitPreservesCommentsAndBlockScalars(t *testing.T) {
	tmpl := `values:
  - name: Name
    type: string
    required: true
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Name }}
data:
  # a comment that must survive
  script.sh: |
    echo one
    ---
    echo two
---
apiVersion: v1
kind: Service
metadata:
  name: {{ .Name }}
`
	dir := t.TempDir()
	if err := generate(t, tmpl, []string{"-Name=demo", "-o", dir}, ""); err != nil {
		t.Fatalf("generate: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "demo-configmap.yaml"))
	if !strings.Contains(string(got), "# a comment that must survive") {
		t.Errorf("comments were dropped during splitting:\n%s", got)
	}
	if !strings.Contains(string(got), "    ---\n    echo two") {
		t.Errorf("an indented --- inside a block scalar was treated as a separator:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "demo-service.yaml")); err != nil {
		t.Errorf("the second document was not written: %v", err)
	}
}

func TestIsZeroQuantityHelper(t *testing.T) {
	isZero, ok := templateFuncs["isZero"].(func(interface{}) bool)
	if !ok {
		t.Fatal("isZero is not registered")
	}
	for _, zero := range []interface{}{"", "0", "0m", "0Mi", 0, float64(0), nil, []interface{}{}} {
		if !isZero(zero) {
			t.Errorf("isZero(%#v) = false, want true", zero)
		}
	}
	for _, nonZero := range []interface{}{"100m", "1", "256Mi", 5, float64(1.5), []interface{}{"a"}} {
		if isZero(nonZero) {
			t.Errorf("isZero(%#v) = true, want false", nonZero)
		}
	}
}

func TestRepoOnly(t *testing.T) {
	repoOnly, ok := templateFuncs["repoOnly"].(func(string) string)
	if !ok {
		t.Fatal("repoOnly is not registered")
	}
	cases := map[string]string{
		"opencloudeu/opencloud:9.1.1":  "opencloudeu/opencloud",
		"ghcr.io/a/b:1.0":              "ghcr.io/a/b",
		"registry:5000/team/app:1.2.3": "registry:5000/team/app",
		"registry:5000/team/app":       "registry:5000/team/app",
		"nginx":                        "nginx",
	}
	for in, want := range cases {
		if got := repoOnly(in); got != want {
			t.Errorf("repoOnly(%q) = %q, want %q", in, got, want)
		}
	}
}
