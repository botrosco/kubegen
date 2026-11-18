# Kubegen

**Kubegen** is a lightweight CLI tool written for generating Kubernetes manifests from templates.

It reads custom `.gotmpl` files that contain YAML frontmatter to define expected variables. Based on this frontmatter, `kubegen` dynamically generates command-line flags, enforces required fields, handles default values, and outputs the rendered templates.

## Features

* **Dynamic CLI Flags:** Automatically generates command-line flags (with type enforcement for `string`, `bool`, and `int`) based on your template's YAML frontmatter.
* **Values Files:** Supply any value — including lists and maps, which flags cannot express — from a YAML or JSON document with `-f values.yaml` (or `-f -` for stdin).
* **Embedded Templates:** Bundle your organization's standard templates directly inside the compiled binary for easy, single-file distribution.
* **Local Overrides:** Reference local files on your filesystem to override embedded templates or test new ones.
* **Built-in Validation:** Enforces `required` fields after the merge, and rejects a values-file key the template does not declare, so a typo cannot silently fall back to a default.
* **Intelligent Output:** Print to standard output, or provide a target directory to automatically split multi-document templates into individually named files (`<name>-<kind>.yaml`).
* **Self-Documenting:** Use the `info` command to instantly see what variables a template expects, and which of them need a values file.

---

## Installation

1. Download latest build for your platform from [releases](/releases/tag/latest)
2. *(Optional)* Move the binary to your PATH e.g. for linux:
   ```bash
   mv kubegen_v0.1.0_linux /usr/local/bin/kubegen
   ```

---

## Build

### Prerequisites
* Go 1.26 or higher

### Setup
1. Clone or initialize the repository.
2. Download the required YAML dependency:
   ```bash
   go get gopkg.in/yaml.v3
   ```
3. Build the binary from the `cmd` directory:
   ```bash
   go build -o kubegen ./cmd/kubegen
   ```
4. *(Optional)* Move the binary to your PATH:
   ```bash
   mv kubegen /usr/local/bin/
   ```

---

## Directory Structure

For the embedded templates to work correctly, your project should look like this before building:

```text
kubegen/
├── go.mod
├── go.sum
├── cmd/
│   └── kubegen/
│       └── main.go           # CLI entry point
└── pkg/
    ├── tpl/
    │   ├── tpl.go
    │   └── templates/        # Put your .gotmpl files here
    │       └── deployment.gotmpl
    ├── parser/
    │   └── parser.go
    └── generator/
        └── generator.go
```

---

## Template Format

Templates must contain a YAML frontmatter block defining the variables, followed by a `---` separator, and then the standard Go `text/template` body.
Also see pre-packaged templates: [pkg/tpl/templates](pkg/tpl/templates)

**Example (`templates/deployment.gotmpl`):**
```yaml
values:
  - name: Name
    type: string
    description: Name of the application
    required: true
  - name: Replicas
    type: int
    description: Number of pod replicas
    default: 3
    required: false
  - name: UseGpu
    type: bool
    description: Whether to request GPU resources
    default: false
    required: false
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Name }}
spec:
  replicas: {{ .Replicas }}
  template:
    spec:
      containers:
      - name: {{ .Name }}
        image: nginx:latest
        {{- if .UseGpu }}
        resources:
          limits:
            [nvidia.com/gpu](https://nvidia.com/gpu): 1
        {{- end }}
```

### Supported Data Types
* `string`, `int`, `bool` — supplied by flag, values file, or default
* `list`, `map`, `object` — supplied by values file only (no flag is created)

An unset `type` defaults to `string`. See `kubegen info <template>` for a `SOURCE`
column that says which input mode each value needs.

### Template helpers

The body may use these functions in addition to the `text/template` builtins:

| Function | Purpose |
|---|---|
| `isZero` | true for `""`, `0`, `0m`, `0Mi`, `0Gi`, `0M`, `0G`, empty lists/maps — use it to omit values nobody chose |
| `hasKey` | whether a map value supplied a key, so a deliberate `0` is not mistaken for absent |
| `repoOnly` | strip a tag from an image reference (`ghcr.io/a/b:1.2.3` → `ghcr.io/a/b`), keeping registry ports |
| `quote` / `squote` | YAML double- / single-quoted rendering (single quotes keep regexes readable) |
| `default` | fall back when a value is empty |
| `indent`, `join`, `lower` | the usual string helpers |
| `fail` | abort the render with a message, for template-side validation |

---

## Usage

### 1. List Bundled Templates
See which templates are compiled into the binary.
```bash
kubegen list
```

### 2. View Template Info
Check the required flags, types, and defaults for a specific template. You can use the short name for embedded templates or provide a path to a local file.
```bash
kubegen info deployment
```
**Output:**
```text
Available Values:
--------------------------------------------------------------------------------
NAME        TYPE      REQUIRED   DEFAULT   DESCRIPTION
Name        string    true       -         Name of the application
Replicas    int       false      3         Number of pod replicas
UseGpu      bool      false      false     Whether to request GPU resources
```

### 3. Generate Manifests
Generate the final text by passing the required variables as CLI flags. By default, this prints to standard output.
```bash
kubegen generate deployment --Name=my-web-app --Replicas=5 --UseGpu=true
```

#### Values files
Anything non-trivial is easier as a document. `-f`/`--values` takes a YAML file
(JSON works too, since YAML is a superset) and `-f -` reads stdin:

```bash
kubegen generate app -f values.yaml -o ./manifests
```

```yaml
# values.yaml -- keys are template value names
Image: ghcr.io/example/app:1.4.2
TargetPort: 8080
Hostnames: ["app.rossd.net"]
Volumes:
  - {name: app-data, mountPath: /var/lib/app, hostPath: /opt/apps/app/data}
Env:
  - {name: APP_URL, value: "https://app.rossd.net"}
  - {name: APP_TOKEN, secretName: app-secret, secretKey: TOKEN}
```

**Precedence: frontmatter `default` → values file → explicit flag.** A flag only wins
when it is actually passed, so a template default never shadows the file. A key the
template does not declare is an error (typo protection), and `required` is checked
after the merge.

#### Saving to a Directory (Intelligent Splitting)
Use the `-o` or `--output` flag to specify a target directory. Kubegen splits the
render into documents and writes one file per object, named
`<metadata.name>-<kind>.yaml` with the kind lowercased — uniformly, for every kind.

```bash
kubegen generate app -f values.yaml -o ./manifests
```
**Output:**
```text
Writing manifests to directory: ./manifests
  - Created: my-app-deployment.yaml
  - Created: my-app-service.yaml
  - Created: my-app-secret.yaml
```

A template may choose its own file name (a `kustomization.yaml` must not become
`my-app-kustomization.yaml`) with an annotation, which is stripped from the written
file:

```yaml
metadata:
  name: my-app
  annotations:
    kubegen.io/filename: kustomization.yaml
```

Splitting is line-based on a `---` separator line, so comments survive and an
indented `---` inside a block scalar is not treated as a boundary.

#### Breaking changes in this version

Two defaults changed so generated output matches a gitops repository that declares the
namespace once and lists every object:

* **File naming:** a Deployment is now `<name>-deployment.yaml` (was `<name>.yaml`).
  Every other kind already used the `<name>-<kind>.yaml` form.
* **Namespace:** `metadata.namespace` is no longer emitted by default. Pass
  `-EmitNamespace=true` (or `EmitNamespace: true` in the values file) to restore it.
  The `Namespace` value is always available to the template for cross-references.

---

## Bundled templates

| Template | What it emits |
|---|---|
| `app` | the whole bundle: Deployment, Service, HTTPRoute, optional `-ext` SSO route, `ImageRepository`, optional `ImagePolicy`, placeholder Secret, and the app folder's `kustomization.yaml` — driven by a values file |
| `workload` | a Deployment with its Service, optional route(s), ConfigMap, Secret and an optional companion DB Deployment |
| `external` | `EndpointSlice` + mirror-skipping Service + route for an application already running on another host |
| `helm` | `HelmRepository` + `HelmRelease`, with an optional pinned chart version |
| `namespace` | a Namespace plus the folder `kustomization.yaml`, including the image-automation component patch |

Run `kubegen info <template>` for the exact values, defaults and input modes.
