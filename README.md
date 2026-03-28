# Kubegen

**Kubegen** is a lightweight, dynamically-typed CLI tool written in Go for generating Kubernetes manifests (or any text-based files) from packaged Go templates. 

It reads custom `.gotmpl` files that contain YAML frontmatter to define expected variables. Based on this frontmatter, `kubegen` dynamically generates command-line flags, enforces required fields, handles default values, and outputs the rendered template.

## ✨ Features

* **Dynamic CLI Flags:** Automatically generates command-line flags (with type enforcement for `string`, `bool`, and `int`) based on your template's YAML frontmatter.
* **Embedded Templates:** Bundle your organization's standard templates directly inside the compiled binary for easy, single-file distribution.
* **Local Overrides:** Reference local files on your filesystem to override embedded templates or test new ones.
* **Built-in Validation:** Enforces `required` fields before attempting to render the template.
* **Self-Documenting:** Use the `info` command to instantly see what variables a template expects.

---

## Installation & Build

### Prerequisites
* Go 1.16 or higher (for `embed` support)

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
    │   └── templates/        # Put your .gotmpl files here before building!
    │       └── deployment.gotmpl 
    ├── parser/
    │   └── parser.go         
    └── generator/
        └── generator.go      
```

---

## 📄 Template Format

Templates must contain a YAML frontmatter block defining the variables, followed by a `---` separator, and then the standard Go `text/template` body.

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
  - name: Image
    type: string
    description: Image of application
    required: true
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
        image: {{ .Image }}
```

### Supported Data Types
* `string`
* `int`
* `bool`

---

## Usage

### 1. List Bundled Templates
See which templates are compiled into the binary.
```bash
kubegen list
```
**Output:**
```text
Bundled Templates:
  - deployment
  - service
```

### 2. View Template Info
Check the required flags, types, and defaults for a specific template. You can use the short name for embedded templates or provide a path to a local file.
```bash
./kubegen info deployment
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
Generate the final text by passing the required variables as CLI flags.
```bash
kubegen generate deployment --Name=my-web-app --Replicas=5 --UseGpu=true
```

To save the output to a file, simply redirect standard output:
```bash
kubegen generate deployment --Name=my-web-app > my-web-app-deployment.yaml
```
