# Spec Delta

## Purpose

Gives templates the values they need to describe a real workload — routing, volumes,
environment, container overrides, chart and external-host parameters — and makes the
emitted output obey a consuming repository's conventions instead of hardcoded defaults.

## ADDED Requirements

### Requirement: Output file names are uniform

Every rendered document SHALL be written as `<metadata.name>-<kind>.yaml` with the kind
lowercased, with no special case for any kind.

#### Scenario: A Deployment is emitted

- **WHEN** a template renders a `Deployment` named `app`
- **THEN** the file is `app-deployment.yaml`

#### Scenario: Several documents are emitted

- **WHEN** a template renders a Deployment, a Service and an HTTPRoute named `app`
- **THEN** the files are `app-deployment.yaml`, `app-service.yaml` and `app-httproute.yaml`

### Requirement: metadata.namespace is emitted only on request

Templates SHALL emit `metadata.namespace` only when the `EmitNamespace` value is true,
which SHALL default to false. The namespace value MUST remain available to the template
for cross-references.

#### Scenario: Default invocation

- **WHEN** a workload is generated without setting `EmitNamespace`
- **THEN** no emitted object carries `metadata.namespace`

#### Scenario: Namespace still usable for references

- **WHEN** `EmitNamespace` is false and the template needs the namespace in a sourceRef or backendRef
- **THEN** the value is still available and renders where the template places it

### Requirement: Routing values are supplied, not invented

Hostnames and route parents SHALL come from template values. A route MUST NOT be
rendered with a hostname the template invented, and the default parent set SHALL be
expressed as a value with a default rather than hardcoded in the template body.

#### Scenario: Hostname is provided

- **WHEN** the values give one hostname
- **THEN** the route lists exactly that hostname

#### Scenario: External SSO route is requested

- **WHEN** `CreateExternalRoute` is true
- **THEN** an additional route named `<name>-ext` is emitted with the `gateway-external`
  parent and a backendRef to the configured external proxy service, while the primary
  route keeps its own parents and the application Service

#### Scenario: External route is not requested

- **WHEN** `CreateExternalRoute` is false
- **THEN** only the primary route is emitted

### Requirement: Volumes are declared as data with an explicit type

Templates SHALL accept a list of volumes, each with a name, a container mount path, a
hostPath path and a hostPath `type`. The `type` SHALL always be rendered, defaulting to
`DirectoryOrCreate`, and a claim MUST NOT be introduced by this path.

#### Scenario: Two volumes are declared

- **WHEN** the values declare two hostPath volumes
- **THEN** the container gets both volumeMounts and the pod gets both volumes, each with its `type`

#### Scenario: Volume type is stated

- **WHEN** a volume declares `type: Directory`
- **THEN** the rendered hostPath carries `Directory` rather than the default

### Requirement: Environment and container overrides are declarable

Templates SHALL accept an environment list whose entries are either a literal value or
a secret key reference, plus `command` and `args` as string lists. An empty list MUST
render nothing rather than an empty YAML block.

#### Scenario: Environment mixes literal and secret entries

- **WHEN** the values declare one literal env var and one secret-backed env var
- **THEN** both render, the secret one as a `valueFrom.secretKeyRef`

#### Scenario: Command is overridden

- **WHEN** the values declare `command` and `args`
- **THEN** the container renders them in place of the image's entrypoint

#### Scenario: No env is declared

- **WHEN** the environment list is empty
- **THEN** no `env` key appears in the rendered container

### Requirement: Zero quantities are omitted, not rendered

A resource request or limit whose value is empty or zero (`0m`, `0`, `0Mi`) SHALL be
omitted, and a `resources` block with nothing left SHALL be omitted entirely.

#### Scenario: Default CPU request

- **WHEN** no CPU request is chosen
- **THEN** the rendered container carries no `cpu` request, rather than `0m`

#### Scenario: No resources at all

- **WHEN** no resource value is set
- **THEN** the container renders without a `resources` block

### Requirement: Chart and external-host parameters are values

The `helm` template SHALL accept a chart version and render it when provided. The
`external` template SHALL accept the endpoint address and the hostnames instead of
hardcoding them.

#### Scenario: Chart version is bounded

- **WHEN** the `helm` template is given a version
- **THEN** the HelmRelease renders `spec.chart.spec.version` with that value

#### Scenario: External host address

- **WHEN** the `external` template is given an address
- **THEN** the EndpointSlice carries that address and no other

### Requirement: Secret placeholders cover the declared keys

Templates SHALL emit a placeholder Secret whose keys come from a declared list, so a
workload with several credentials does not need its Secret written by hand. Rendered
values MUST be obvious placeholders; encryption remains the consumer's step.

#### Scenario: Two secret keys

- **WHEN** the values declare two secret key names
- **THEN** the rendered Secret has both keys, each with a placeholder value

#### Scenario: No secret keys

- **WHEN** no secret key is declared
- **THEN** no Secret is rendered
