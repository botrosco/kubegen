# Spec Delta

## Purpose

Generates a namespace together with the kustomization and image-automation wiring a
consuming repository requires, so a new namespace is one invocation instead of one
object plus hand-written scaffolding.

## ADDED Requirements

### Requirement: A namespace render includes its kustomization

The `namespace` template SHALL emit the Namespace object and a `kustomization.yaml` for
that namespace folder that sets `namespace:`, references the namespace object, and
carries the image-update-automation component.

#### Scenario: New namespace is generated

- **WHEN** the `namespace` template is generated for `demo`
- **THEN** the output contains a Namespace named `demo` and a `kustomization.yaml` with `namespace: demo`

### Requirement: The automation patch is emitted with an explicit update path

The kustomization SHALL patch the inherited image-update-automation object's
`spec.update.path` to the generated namespace's own path, and `spec.interval` to a value
supplied by the caller. The interval MUST NOT have a default.

#### Scenario: Update path is scoped

- **WHEN** the namespace template generates for `demo`
- **THEN** the patch sets `spec.update.path` to `./live/demo`

#### Scenario: Interval is required

- **WHEN** no interval is supplied
- **THEN** generation fails rather than emitting a colliding default

### Requirement: Component path is a value

The path to the image-update-automation component SHALL be a template value with a
sensible default, so a repository with a different layout can override it.

#### Scenario: Different component location

- **WHEN** the caller overrides the component path
- **THEN** the kustomization references that path instead of the default

### Requirement: Generated namespace output obeys the shared conventions

The namespace render SHALL follow the same output rules as every other template: uniform
file naming, and no `metadata.namespace` on the Namespace object unless requested.

#### Scenario: Files are named

- **WHEN** the namespace render completes
- **THEN** the files are `<name>-namespace.yaml` and `kustomization.yaml`
