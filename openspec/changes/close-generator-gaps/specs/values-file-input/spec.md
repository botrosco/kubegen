# Spec Delta

## Purpose

Lets a template declare and receive structured values (lists, maps, nesting) from a
values document, so one invocation can describe a whole workload instead of dozens of
stringly-typed flags.

## ADDED Requirements

### Requirement: Values are supplied as a YAML or JSON document

`kubegen generate` SHALL accept `-f <file>` / `--values <file>` naming a YAML document
whose keys are template value names, and `-f -` SHALL read that document from stdin.
The document MAY be JSON, since YAML is a superset of it.

#### Scenario: Values file supplies a list

- **WHEN** a template declares `Hostnames` of type `list` and the values file supplies three hostnames
- **THEN** the render receives all three, and `kubegen info` reports the value as values-file only

#### Scenario: Values file is absent

- **WHEN** no `-f` is given
- **THEN** behaviour is unchanged from flag-and-default only, and no error is raised for templates that declare no non-scalar values

#### Scenario: Values file cannot be read or parsed

- **WHEN** the named file does not exist, or does not parse as YAML/JSON
- **THEN** generation fails with an error naming the file and the parse problem, and nothing is written

### Requirement: Precedence is default, then values file, then flag

For every value the merge order SHALL be: frontmatter `default`, then the values file,
then a flag that was explicitly provided on the command line. A flag present with its
default value MUST NOT override a values-file entry.

#### Scenario: Both sources provide a value

- **WHEN** the values file sets `Image` and the command line also passes `-Image=…`
- **THEN** the flag wins, because it was explicitly provided

#### Scenario: Only the default is left

- **WHEN** neither the values file nor the command line provides a value that has a frontmatter default
- **THEN** the default is used

### Requirement: Non-scalar values are declared in frontmatter and never become flags

Frontmatter `type` SHALL accept `list`, `map` and `object` in addition to `string`,
`int` and `bool`. Values of the non-scalar types SHALL be supplied only through the
values file, and MUST NOT create command-line flags.

#### Scenario: A list value is requested by flag

- **WHEN** the user passes `-Hostnames=a,b` for a `list` value
- **THEN** the flag is not defined, and the error names the undefined flag

#### Scenario: info documents the input mode

- **WHEN** `kubegen info <template>` is run for a template with a `list` value
- **THEN** the output marks that value as supplied by a values file

### Requirement: Unknown values-file keys are rejected

A values-file key that the template does not declare SHALL be an error, so a typo
cannot silently fall back to a default.

#### Scenario: Value name is misspelled

- **WHEN** the values file contains `Hosntames`
- **THEN** generation fails and names the unknown key together with the template that rejected it

### Requirement: Required values are enforced after the merge

A value marked `required` SHALL be satisfied by either source, and SHALL fail only when
neither the values file nor an explicit flag provides it.

#### Scenario: Required value supplied by the values file

- **WHEN** a required value is present in the values file and no flag is passed
- **THEN** generation proceeds

#### Scenario: Required value missing everywhere

- **WHEN** a required value is absent from both sources and has no default
- **THEN** generation fails with an error naming the value
