# Proposal

## Why

The `k8s-manifests` repo's `create-workload` procedure is generator-first, but the
installed kubegen only covers part of a workload: re-implementing `opencloud` through
that procedure produced **79 generated lines against 219 accepted lines** — roughly
64% hand-written — because kubegen cannot express lists or maps at all (no env list,
no second volume, no `command`/`args`), emits a shape that breaks the repo's
conventions (`metadata.namespace`, `<name>.yaml`, internal-only route, plaintext
secret, hostPath without `type`), and has no object for image tracking or a bundle.

Sixteen gaps are recorded, verified against the installed binary, in the
`k8s-manifests` register
(`.agents/skills/create-workload/references/generator-gaps.md`). Every one of them is
work that kubegen should own and every one is currently hand-written per workload, so
the drift the procedure was written to stop keeps coming back through the hand steps.

## What Changes

- **Values-file input (the blocker, done first).** `kubegen generate <tpl> -f values.yaml`
  (also `-f -` for stdin) merges a YAML/JSON document into the render, and template
  frontmatter gains `list`, `map` and `object` types so a template can declare the
  values that need them. Precedence = frontmatter default → values file → flag.
  Scalar types keep their flags; non-scalar types have no flag.
- **Template vocabulary.** New variables the existing templates were missing:
  hostnames, gateway parents, an external (SSO-proxy) route, env lists, several
  volumes with a hostPath `type`, container `command`/`args`, chart `version`, the
  external-host address, and image-policy marker support.
- **Output conventions (BREAKING).** Files are named `<name>-<kind>.yaml` uniformly,
  including Deployments, and `metadata.namespace` is emitted only when a template asks
  for it (`EmitNamespace`, default false) — the consuming repo declares the namespace
  once in its kustomization.
- **New `app` template.** One invocation emits the whole bundle this repo uses:
  Deployment, Service, internal HTTPRoute, optional `-ext` SSO route,
  `ImageRepository` + `ImagePolicy`, placeholder Secret, and the app-folder
  `kustomization.yaml`.
- **`namespace` template grows up.** It emits the namespace kustomization with the
  image-automation component patch (update path + interval) it currently omits.
- **Zero-resource fix.** A `0m` CPU request is omitted instead of rendered, so a
  default run cannot emit a value nobody chose.
- **Tests.** First `_test.go` files in the repo: parser/generator unit tests and a
  golden test for the `app` template.

## Capabilities

### New Capabilities

- `values-file-input`: supplying template values as a YAML/JSON document, including
  lists and maps, merged predictably with flags and frontmatter defaults.
- `template-vocabulary`: the value vocabulary a template can declare — routing,
  volumes, environment, container overrides, chart and external-host parameters, and
  the resource-emission rules.
- `app-bundle`: the single-invocation application bundle (workload, routing, image
  tracking, secret placeholder, folder kustomization).
- `namespace-scaffolding`: generating a namespace together with the kustomization and
  image-automation patch a consuming repo needs.

### Modified Capabilities

None. `openspec/specs/` is empty: this is the repository's first OpenSpec change.

## Impact

- **BREAKING for existing users of the bundled templates**: Deployment files are now
  `<name>-deployment.yaml` (was `<name>.yaml`), and `metadata.namespace` is no longer
  emitted by default (pass `-EmitNamespace=true` to keep the old behaviour). Both are
  required by the consuming repo's conventions; both are documented in `README.md`.
- Code: `pkg/parser` (frontmatter types, values-file parsing), `pkg/generator`
  (merge, flag generation, naming, namespace handling), `pkg/tpl/templates/*.gotmpl`
  (four existing templates reworked, one new `app.gotmpl`), `cmd/kubegen` (nothing
  beyond wiring), `README.md`.
- New `_test.go` files under `pkg/parser`, `pkg/generator` and a template golden test.
- No new dependencies: `gopkg.in/yaml.v3` already parses both YAML and the JSON subset
  the values file needs.
- Consumer: the `k8s-manifests` `create-workload` skill, whose register entries close
  as each gap lands. Nothing in that repo is edited by this change.
