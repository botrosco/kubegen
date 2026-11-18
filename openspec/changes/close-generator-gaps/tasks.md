# Tasks

## 1. Values-file input (the blocker — do first)

- [x] 1.1 Extend `pkg/parser` with the `list`, `map` and `object` types: `ValueDef` keeps its fields, `info` reporting gains the input mode. Unit-test the parse of a frontmatter declaring each type.
- [x] 1.2 Implement `-f/--values <file>` and `-f -` (stdin) in `pkg/generator`: parse the document, reject keys the template does not declare, and merge `default → values file → explicit flag` using `fs.Visit` for explicitness.
- [x] 1.3 Apply the merged values to every type — scalars from either source, non-scalars only from the values file — and enforce `required` after the merge. Verify the negative paths: unknown key, unreadable file, missing required value, list value requested as a flag.
- [x] 1.4 Add `pkg/generator/generator_test.go` covering precedence, unknown-key rejection, required-after-merge, and non-scalar delivery.

## 2. Output conventions (breaking)

- [x] 2.1 Make splitting uniform: `<metadata.name>-<kind>.yaml` for every kind, no Deployment special case. Verify the probe render writes `app-deployment.yaml` and not `app.yaml`.
- [x] 2.2 Add `EmitNamespace` (bool, default false) to the workload, external, helm and namespace templates, gating every `metadata.namespace` while keeping the namespace available for cross-references. Verify a default render carries no namespace and `-EmitNamespace=true` restores it.
- [x] 2.3 Omit zero/empty resource quantities and empty list-driven blocks (env, volumes, hostnames, resources) in the shared templates. Verify a default invocation renders no `cpu: 0m` and no empty `resources`/`env` blocks.
- [x] 2.4 Update `README.md` for the breaking changes and the migration note (`-EmitNamespace=true` for the old namespace behaviour; the Deployment file rename).

## 3. Template vocabulary

- [x] 3.1 Routing values: `Hostnames` (list), `GatewayParents` (list of name/namespace/sectionName, defaulting to `gateway`/`istio-system`/`https`), and `CreateExternalRoute` + `ExternalBackendName`/`ExternalBackendNamespace`/`ExternalBackendPort` emitting a `<name>-ext` route. Verify both shapes render — one dual-parent route, and the two-route SSO pair.
- [x] 3.2 Workload data: `Volumes` (list with an explicit `type`, default `DirectoryOrCreate`), `Env` (list of literal or secretKeyRef entries), `Command`/`Args` (string lists), `SecretKeys` (list → placeholder `stringData`). Verify a render carrying two volumes, a literal and a secret env var, and a command override.
- [x] 3.3 Chart and external parameters: a `Version` value on `helm` rendered as `spec.chart.spec.version`; `Address` and `Hostnames` on `external` replacing the hardcoded `192.168.1.5` and `<Name>.rossd.net`. Verify both renders show the supplied values and no hardcoded ones remain in the templates.
- [x] 3.4 Image-policy marker support: an `ImagePolicy` value name rendered as the `{"$imagepolicy": …}` setter comment on the image line, present only when a policy exists.

## 4. The app bundle template

- [x] 4.1 Create `pkg/tpl/templates/app.gotmpl`: frontmatter declaring the container, volumes, env, `command`/`args`, routing, secret keys, image-policy shape and kustomization values. Render Deployment, Service and internal HTTPRoute first.
- [x] 4.2 Add the optional `-ext` route, the placeholder Secret, the `ImageRepository` and the three `ImagePolicy` shapes (semver, numerical-with-extract, none). Verify each shape renders and that the marker appears exactly when a policy does.
- [x] 4.3 Emit the app-folder `kustomization.yaml` listing exactly the files written, sorted, with no `namespace:` field.
- [x] 4.4 Support an init container as data, rendered ahead of the main container. Verify an opencloud-shaped values file (init chown, two volumes, 21 env vars, command override, two secret-free containers) renders end to end.

## 5. Namespace scaffolding

- [x] 5.1 Extend `namespace.gotmpl` to emit the kustomization with `namespace:`, the component path, and the two-operation patch; add `ComponentPath`, `UpdatePath` and a required `Interval`. Verify a render for a scratch namespace and that a missing interval fails.
- [x] 5.2 Golden test `pkg/tpl/templates/` for the namespace render.

## 6. Tests and verification

- [x] 6.1 `pkg/tpl/templates/app_golden_test.go` with `testdata/app-values.yaml` and `testdata/app.golden.yaml`: render the bundle and compare byte-for-byte.
- [x] 6.2 Full verification: `go build`, `gofmt -l .` clean, `go vet ./...`, `go test ./...` green, plus the existing smoke test (`list`, `info workload`, generate with `-o`, and the required-flag negative path).
- [x] 6.3 Regenerate the `k8s-manifests` `opencloud` workload from the new `app` template and diff it against the deployed manifests; report the remaining hand steps (expected: sops encryption only) as the closure evidence for the register.
- [x] 6.4 Commit with `jj` and push `main` to Forgejo; state in the commit message which register entries close.

## Workflow follow-up

- Remove the closed entries from the `k8s-manifests` register in that repo, and drop the corresponding hand steps from its `SKILL.md` / `verify.md`.
- Re-check the register's remaining items (`ReferenceGrant`/Terraform wiring, registration in the namespace kustomization, sops, resource-targets row) — those stay consumer-side by decision, but the register should say so explicitly once the generator covers everything above.
