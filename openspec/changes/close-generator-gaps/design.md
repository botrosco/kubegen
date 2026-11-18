# Design

## Context

kubegen is a small Go CLI: `pkg/parser` splits YAML frontmatter from a `text/template`
body, `pkg/generator` turns frontmatter value definitions into `flag` flags and renders,
`pkg/tpl` resolves local-or-embedded templates, and `pkg/tpl/templates/*.gotmpl` holds
four bundled templates (`workload`, `external`, `helm`, `namespace`). There are no
tests. Consumers embed it from a devbox toolchain and call it from a procedure.

The sixteen gaps come from the `k8s-manifests` register, all verified against the
installed binary. Their real shape is not "missing features" but **one missing
capability (non-scalar values) plus a vocabulary that assumes a single container, one
volume, one env var and one route**.

Constraints that shape this:

- The consumer repo's conventions are strict: no `metadata.namespace`, one object per
  file, `<name>-<kind>.yaml`, every file listed in a kustomization, images tracked with
  the Flux `{"$imagepolicy": ...}` setter marker, hostPath storage with an explicit
  `type`, secrets encrypted outside kubegen.
- Two consumers of one binary (the repo's own manifests and any future one) mean the
  bundle's repo-specific pieces must be **data**, not baked-in strings.
- kubegen has no `go` on the host PATH; builds go through the devbox toolchain, so the
  loop must stay quick and local (`go build`, `go test`, `gofmt`, `go vet`).

## Goals / Non-Goals

**Goals**

- Any workload the consumer repo needs can be emitted without hand-writing a manifest
  that kubegen could have produced — measured against `opencloud`, the worst case
  (21 env vars, two containers, two volumes, command override, two probes).
- A values file is the primary input for anything non-trivial; flags stay for small
  scalar-only templates.
- Output obeys the consumer's conventions by default, with the two behaviour changes
  documented as breaking.

**Non-Goals**

- Encrypting secrets (sops is a consumer-side step), or applying anything to a cluster.
- Becoming a general Kubernetes scaffolding tool: only the shapes the consumer repo
  uses get templates.
- Backward compatibility with the old file naming/namespace emission beyond a
  documented `EmitNamespace` opt-in and a README migration note.

## Decisions

### D1 — Values file first, and it defines the ordering of everything else

`-f/--values <file>` (plus `-f -` for stdin) loads a YAML document whose keys are
template value names. `flag` still parses the CLI; the merge happens after parsing:

```
frontmatter default  →  values file  →  explicit flag
```

An explicit flag wins only when it was actually set (`fs.Visit`), which is already how
required-flag detection works. Values files are parsed with `yaml.v3`, which handles the
JSON subset too — no new dependency.

Rationale: every remaining gap is a list or a map (hostnames, env, volumes, parents,
secret keys, policy filters). A template that cannot express two hostnames cannot
express a bundle, so doing the bundle first would mean rewriting it. The measured
64% hand-written share of `opencloud` is essentially all of this one gap.

*Alternatives:* JSON-only values file (rejected: the consumer repo is all YAML);
repeated flags (`-Env A=1 -Env B=2`) — rejected: no nesting, no volumes, and the flag
help becomes noise; environment variables — rejected: unusable for structured values.

### D2 — New frontmatter types: `list`, `map`, `object`

`type` gains `list` (sequence), `map`/`object` (mapping). Only `string|int|bool` create
flags; non-scalar values must arrive from a values file and are reported as
"values-file only" in `info`. `required` still applies to all types and is checked
after the merge, so a missing list fails with a clear message.

Rationale: keeps the flag surface honest (a list flag would be stringly-typed) and
makes `kubegen info` self-documenting about which values need a file.

*Alternative:* a single `-ValuesJSON` flag carrying the whole document — rejected: two
ways to say everything, and it hides per-value required checks.

### D3 — Uniform file naming, and namespace emission becomes opt-in (BREAKING ×2)

Splitting writes `<name>-<kind>.yaml` for every kind, dropping the Deployment special
case. `metadata.namespace` is emitted only when the template sets `EmitNamespace`
(default `false`); the namespace value stays available for cross-references
(`sourceRef`, backendRefs, `imageRepositoryRef`).

Rationale: the consumer declares the namespace once in a kustomization, so an inline
namespace is a convention violation, and `<name>.yaml` for one kind while everything
else is `<name>-<kind>.yaml` is an inconsistency with no upside. Keeping
`EmitNamespace=true` available means the change is opting-in-able for existing users.

*Alternative:* a `-naming` flag with several conventions — rejected: speculative
generality for a tool with one real consumer.

### D4 — Routing is data: hostnames, parents, and an optional external route

`workload` and `external` gain:

- `Hostnames` (list, required for a route) — replaces the invented `<Name>.rossd.net`.
- `GatewayParents` (list of `{name, namespace, sectionName}`), default one entry:
  `gateway` / `istio-system` / `https`.
- `CreateExternalRoute` (bool) + `ExternalBackendName` / `ExternalBackendNamespace` /
  `ExternalBackendPort`, defaulting to `netbird-reverse-proxy` in `netbird` on `443`.

The external route is emitted as a **second** HTTPRoute named `<name>-ext`, because the
SSO shape needs one route pointing at the proxy on `gateway-external` and one internal
route the proxy dials — two objects, not one with two parents.

Rationale: exposure is a decision the consuming procedure makes; kubegen should not
guess a hostname or a parent, but it also should not make the two-object shape a hand
step. A default parent keeps simple invocations one flag long.

### D5 — Volumes, env, command/args are lists/maps

- `Volumes`: list of `{name, mountPath, hostPath, type, readOnly}` → renders
  `volumeMounts` + `volumes`, always with the hostPath `type` (default
  `DirectoryOrCreate`) and never a claim.
- `Env`: list of `{name, value}` **or** `{name, secretName, secretKey}` → renders
  `env` entries of either shape.
- `Command` / `Args`: string lists.
- `SecretKeys`: list of key names → renders a `stringData` placeholder per key. Values
  stay placeholders; encryption is the consumer's step.

Rationale: these are exactly the `opencloud` hand steps, and each maps to one list.

### D6 — Zero and empty values are omitted, not rendered

`resources` omits a request/limit whose value is empty or `0m`/`0Mi` (a small
`isZeroQuantity` helper) and omits the `resources` block entirely when nothing remains.
Same rule for `hostnames`/`env`/`volumes`: empty lists render nothing rather than an
empty YAML block.

Rationale: the consumer's records table demands deliberate values; a rendered `0m` is
indistinguishable from a decision.

### D7 — `app.gotmpl`: the bundle, driven by a values file

One template, one invocation, emitting Deployment, Service, internal HTTPRoute,
optional `-ext` route, `ImageRepository`, `ImagePolicy`, Secret and the app-folder
`kustomization.yaml`. Its frontmatter declares the container, volumes, env, routing and
image-policy values; everything repo-specific (gateway names/namespace, policy ranges,
kustomization resource list) is a value with a repo-shaped default.

`ImagePolicy` supports the two shapes the consumer uses: `semver` (with a
`filterTags` pattern and a major-bounded range) and `numerical` (with `filterTags` +
`extract` for tags like `build-N`), plus a "no policy" mode that keeps the pinned tag
and the `ImageRepository` only.

The app-folder `kustomization.yaml` is emitted by the same render, listing the objects
that render actually produced, sorted.

Rationale: this is the object the register exists to justify. It must be one review
surface: one invocation, one values file, one diff.

*Alternative:* composable small templates (rejected earlier by the consumer, and it
reintroduces skipped steps).

### D8 — `namespace` gains the kustomization and the automation patch

`namespace` keeps emitting the Namespace object and adds `kustomization.yaml` with
`namespace:`, the `image-update-automation` component path, and the patch setting
`spec.update.path` and `spec.interval`. New values: `ComponentPath` (default
`../../components/image-update-automation`), `UpdatePath` (default `./live/<name>`),
`Interval` (required, no default — the consumer must pick an unused value, since a
default would silently collide).

Rationale: the inherited update path in the consumer's component is a deliberately
invalid sentinel, so an interval with no default forces the deliberate choice the
consumer's convention demands.

### D9 — Tests, at last

`pkg/parser/parser_test.go` (frontmatter types, missing separator, non-scalar types),
`pkg/generator/generator_test.go` (precedence: default < values file < flag; required
after merge; file naming; `EmitNamespace`; zero-quantity omission), and
`pkg/tpl/templates/app_golden_test.go` (render a fixed values file, compare against
`testdata/app.golden.yaml`).

Rationale: the register's "no tests" gap is why the conventions drifted in the first
place; the golden test is also the only practical way to keep one large template
honest.

### D10 — Keep the four template names; keep `info` compatible

`workload`, `external`, `helm`, `namespace` keep their names and existing scalar
values; new values are additive except the two documented breaking changes. `info`
gains a `VALUES FILE` column ("yes"/"-") rather than changing column meanings.

## Risks / Trade-offs

- **A large template is hard to review** → the golden test pins the whole output; the
  values file is the review surface.
- **Breaking naming/namespace silently breaks a consumer's kustomization** → both
  changes are called out in the proposal and `README.md`, and `-EmitNamespace=true`
  restores one of them; the file rename is caught immediately by `kustomize build`
  (a listed path that no longer exists fails the render).
- **Values-file validation is shallow** → unknown keys in a values file are rejected
  (typo protection); wrong types surface at render time as a template error. Deep
  schema validation is out of scope.
- **`Interval` with no default makes `namespace` fail without a flag** → intended; the
  alternative is a colliding interval in a shared branch.
- **Scope creep into a general scaffolding tool** → the Non-Goals and the register are
  the boundary; no template is added for a shape the consumer does not use.

## Migration Plan

Single commit per concern, `main` bumped and pushed to Forgejo with `jj`. Consumers
pick it up on their next invocation; the `k8s-manifests` register entries that this
change closes are removed in that repo (not here). Rollback is reverting the commits.
