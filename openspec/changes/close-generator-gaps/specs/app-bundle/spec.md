# Spec Delta

## Purpose

Emits a complete application — workload, routing, image tracking, secret placeholder
and folder kustomization — from one invocation and one values document, so a workload
is one review surface instead of a sequence of hand steps.

## ADDED Requirements

### Requirement: One invocation emits the whole bundle

The `app` template SHALL emit, from a single invocation: a Deployment, a Service, an
internal HTTPRoute, an optional external route, an `ImageRepository`, an `ImagePolicy`
when tracking is possible, a placeholder Secret when keys are declared, and the app
folder's `kustomization.yaml`.

#### Scenario: Typical application

- **WHEN** the `app` template is generated with a container, one volume, routing and an image policy
- **THEN** all of those objects are written, plus a `kustomization.yaml` that lists exactly the files that were written

#### Scenario: No external exposure

- **WHEN** the application is only reachable on the internal gateway
- **THEN** no `-ext` route is emitted and the kustomization lists only the files produced

### Requirement: The folder kustomization matches what was written

The emitted `kustomization.yaml` SHALL list exactly the files the render produced, in a
stable order, and MUST NOT introduce a `namespace:` line.

#### Scenario: Bundle of six files

- **WHEN** the render produces six object files
- **THEN** the kustomization lists those six and nothing else

#### Scenario: Namespace ownership

- **WHEN** the bundle is emitted for a namespace that is declared by a parent kustomization
- **THEN** the emitted kustomization carries no `namespace:` field

### Requirement: Image tracking is chosen from the tag shape

The bundle SHALL support three tracking shapes: semantic versions with a
major-bounded range, non-semver resolvable tags via a `filterTags` extract and a
numerical policy, and no policy at all (pinned tag plus `ImageRepository` only). The
image field SHALL carry the `{"$imagepolicy": …}` setter marker exactly when a policy is
emitted.

#### Scenario: Semantic versions

- **WHEN** the tag shape is `MAJOR.MINOR.PATCH` and a semver range is given
- **THEN** the `ImagePolicy` filters that shape and applies the range, and the image line carries the marker

#### Scenario: Non-semver tags

- **WHEN** the tags are a numbered build shape
- **THEN** the policy uses `filterTags` with an extract and a numerical policy, and the marker is present

#### Scenario: Untrackable tags

- **WHEN** no policy is requested
- **THEN** an `ImageRepository` is still emitted, no `ImagePolicy` is emitted, and the image keeps the concrete pinned tag without a marker

### Requirement: The bundle obeys the same conventions as the other templates

Every object the bundle emits MUST follow the output conventions: uniform file naming,
no `metadata.namespace` unless requested, explicit hostPath `type`, no zero quantities,
and comments only where they name a hazard.

#### Scenario: Bundle output is inspected

- **WHEN** the bundle's files are reviewed
- **THEN** they carry no inline namespace, no `0m` quantities, and each hostPath declares a `type`

### Requirement: The bundle is driven by a values document

The bundle's non-scalar inputs (containers' env, volumes, hostnames, parents, secret
keys, policy filter) SHALL be supplied through a values file, and its small scalars MAY
be supplied by flag.

#### Scenario: Environment list of any size

- **WHEN** the application needs twenty environment variables
- **THEN** they are supplied in the values file and all render, with no change to the invocation

#### Scenario: Second container

- **WHEN** the application needs an init container performing a fixup
- **THEN** it is declarable as data and renders ahead of the main container
