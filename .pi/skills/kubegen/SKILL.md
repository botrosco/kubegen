---
name: kubegen
description: Generate Kubernetes manifests with the kubegen CLI (.gotmpl templates with YAML frontmatter), author new templates, and improve the kubegen application itself. Use whenever the task involves running kubegen, writing/modifying .gotmpl template files, or changing kubegen's Go code (parser, generator, template registry) — including fixing gaps in kubegen's capability and pushing those improvements as code changes.
---

# kubegen

Kubegen is a lightweight Go CLI that generates Kubernetes manifests from `.gotmpl`
templates. Each template starts with a YAML frontmatter block declaring its
variables; kubegen turns those into CLI flags, enforces required fields, applies
defaults, and renders the Go `text/template` body.

This skill covers two jobs:

1. **Using** kubegen to generate manifests and author templates.
2. **Improving** the kubegen application when a capability gap is identified —
   implementing the fix in the Go source and pushing it.

## Command reference

```
kubegen list                                  # list bundled (embedded) templates
kubegen info <template>                       # show the variables a template expects
kubegen generate <template> [flags] [-o DIR] # render manifests (stdout, or split files into DIR)
```

- `<template>` is a local file path, an embedded shorthand (`workload` →
  `templates/workload.gotmpl`), or an exact embedded filename (`workload.gotmpl`).
  Local files always win over embedded.
- Flags are Go `flag` style: `-Name=value` or `--Name=value`.
- Types: `string`, `bool` (`-Flag=true`, default false), `int` (default 0).
- `-o DIR` / `--output DIR` splits a multi-document render into one file per
  document: `Deployment` → `<name>.yaml`, everything else → `<name>-<kind>.yaml`
  (kind lowercased, e.g. `app-config-cm-configmap.yaml`). Each file is written
  with a leading `---` document separator.
- Without `-o`, the full multi-document YAML stream is printed to stdout.
- Missing required flags abort with `required flag --<Name> is missing`.

### Reserved flags

`output` and `o` are reserved for the output-dir flag; a template variable named
either is rejected at generation time
(`template variable cannot be named 'output' as it is a reserved flag`). Avoid
these names in frontmatter.

## Template format

Frontmatter, a `\n---\n` separator, then a Go `text/template` body:

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
  - name: UseGpu
    type: bool
    description: Whether to request GPU resources
    default: false
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Name }}
spec:
  replicas: {{ .Replicas }}
```

Value fields (see `parser.ValueDef` in `pkg/parser/parser.go`):
`name`, `type` (`string|bool|int`), `description`, `required` (bool), `default`
(any YAML scalar). `default` for an int may arrive as `int` or `float64`; the
generator coerces both. `required` defaults to false.

Bundled templates live in `pkg/tpl/templates/` and are embedded with
`//go:embed templates/*.gotmpl` in `pkg/tpl/tpl.go` — adding a file there is
enough to bundle it (re-run `list` to confirm).

## Building and testing

The host shell has NO `go` — use the devbox toolchain:

```bash
# one-liner (PATH-prefix form)
cd /opt/git/kubegen
PATH="$HOME/devbox/kubegen/.devbox/nix/profile/default/bin:$PATH" \
  go build -o /tmp/kubegen ./cmd/kubegen

# or via devbox run from the devbox project
cd ~/devbox/kubegen && devbox run -- sh -c \
  'cd /opt/git/kubegen && go build -o /tmp/kubegen ./cmd/kubegen'
```

Smoke test after any change:

```bash
/tmp/kubegen list
/tmp/kubegen info workload
/tmp/kubegen generate workload -Name=demo -Namespace=apps -Image=nginx:latest \
  -TargetPort=8080 -CreateLiveness=true -CreateHTTPRoute=true -CreateConfigMap=true \
  -o /tmp/gen-out && ls /tmp/gen-out
# and the negative path:
/tmp/kubegen generate workload -Namespace=apps -Image=x   # must fail: required --Name
```

`gofmt -l .` and `go vet ./...` should be clean before pushing. Note the CI
workflow (`.forgejo/workflows/build.yaml`) builds only on release publish —
push triggers are commented out — so **local verification is the gate**.

## Architecture

- `cmd/kubegen/main.go` — CLI entry point (`list` / `info` / `generate`). The
  stray `main.go` at the repo root is a vestigial standalone version (no `list`,
  no `-o`) and is NOT the build target; the workflow and README build
  `./cmd/kubegen`.
- `pkg/parser/parser.go` — splits frontmatter from body, parses `ValueDef`s.
- `pkg/generator/generator.go` — builds flags from values, enforces required,
  renders the template, and splits/writes output files.
- `pkg/tpl/tpl.go` — template lookup (local → embedded) + `ListEmbedded`.
- `pkg/tpl/templates/*.gotmpl` — bundled templates.
- No `_test.go` files exist yet; the first change that adds tests should add a
  proper `*_test.go` next to the package being changed.

## Improving kubegen (push improvements when a gap is found)

When a task reveals a capability gap or bug (see checklist below), you are
expected to **fix it in the application and push it**, not paper over it. This
repo is jj-managed with a colocated `.git`; the remote is
`ssh://git@ssh-git.rossd.net/rosco/kubegen.git` (Forgejo).

Workflow:

1. **Identify the gap** — reproduce it first (e.g., render the failing template,
   or read the generator code path).
2. **Implement** — keep the existing structure: entrypoint in `cmd/kubegen`,
   logic in `pkg/{parser,generator,tpl}`, templates in `pkg/tpl/templates/`.
   Small focused changes; prefer fixing shared generator/parser logic over
   per-template patches. If CLI behavior changes, update `README.md`.
3. **Verify** — build, gofmt, go vet, and run the smoke test above, including
   the new behavior. Add a `_test.go` covering the gap if the repo has test
   infra for it by then.
4. **Commit & push with jj** (never git directly):

```bash
jj status                                  # confirm dirty working copy
jj commit -m "fix: <what and why>"
jj bookmark move main --to @-              # main does NOT auto-advance
jj git push -b main                        # pushes bookmark to Forgejo origin
```

If a follow-up needed commit exists after pushing, repeat the move+push.

### Known gaps & candidate improvements (check these first)

- **Vestigial root `main.go`** — duplicate old CLI at repo root; candidate for
  removal (README/workflow/`cmd/kubegen` are canonical).
- **`requried` typos** — `workload.gotmpl` (DBImage/DBTargetPort/DBContainerPath/
  DBCreateSecret) and `helm.gotmpl` use `requried:`; the parser only reads
  `required`, so those get `required=false` (intended, but the typo means any
  future intent to require them silently won't take effect).
- **Liveness probe with no port** — `CreateLiveness=true` without `TargetPort`
  renders `port: 0` (invalid-ish probe). Should be gated on `TargetPort != 0`
  like the Service/HTTPRoute blocks.
- **Hardcoded references** — HTTPRoute templates hardcode parentRef
  `istio-system/gateway` (sectionName `https`) and hostname `<Name>.rossd.net`;
  `external.gotmpl` hardcodes the EndpointSlice address `192.168.1.5`. These
  should be template variables.
- **Multi-doc split** — output splitting is `strings.Split(rendered, "\n---")`;
  a body containing a literal `\n---` line breaks it. Also no `---`-variant
  handling for the first doc.
- **No values-file input** — no `-f values.yaml` / JSON/TOML input mode; more
  than ~20 variables makes flag-style invocation unwieldy.
- **No tests** — zero `_test.go` files; generator/parser logic is untested.
- **Resource blocks** — `workload.gotmpl` emits `cpu: 0m` for requests when any
  resource flag is set (CpuRequest default is `0m`, so requesting "no cpu" still
  renders `0m`); consider omitting empty/zero values via conditional blocks.
- **Toleration rigidity** — tolerations only render when BOTH key and value are
  set; no operator/effect/tolerationSeconds flexibility.