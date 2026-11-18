package parser

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Scalar types are supplied by flag or values file; non-scalar types are
// values-file only (a list cannot be expressed as a single flag).
const (
	TypeString = "string"
	TypeInt    = "int"
	TypeBool   = "bool"
	TypeList   = "list"
	TypeMap    = "map"
	TypeObject = "object"
)

// ValueDef represents a single variable definition from the YAML frontmatter
type ValueDef struct {
	Name        string      `yaml:"name"`
	Type        string      `yaml:"type"`
	Description string      `yaml:"description"`
	Required    bool        `yaml:"required"`
	Default     interface{} `yaml:"default"`
}

type Frontmatter struct {
	Values []ValueDef `yaml:"values"`
}

// IsScalar reports whether the value type can be provided by a command-line flag.
func (v ValueDef) IsScalar() bool {
	switch v.Type {
	case TypeString, TypeInt, TypeBool:
		return true
	}
	return false
}

// KnownType reports whether the value type is one this parser understands.
func KnownType(t string) bool {
	switch t {
	case TypeString, TypeInt, TypeBool, TypeList, TypeMap, TypeObject:
		return true
	}
	return false
}

// Parse splits the content and parses the YAML frontmatter
func Parse(content string) ([]ValueDef, string, error) {
	parts := strings.SplitN(content, "\n---\n", 2)
	if len(parts) < 2 {
		return nil, "", fmt.Errorf("invalid template format: missing '\\n---\\n' separator")
	}

	yamlPart := parts[0]
	templatePart := parts[1]

	var fm Frontmatter
	err := yaml.Unmarshal([]byte(yamlPart), &fm)
	if err != nil {
		return nil, "", fmt.Errorf("error parsing YAML: %w", err)
	}

	for i, v := range fm.Values {
		if v.Name == "" {
			return nil, "", fmt.Errorf("frontmatter value %d has no name", i+1)
		}
		if v.Type == "" {
			fm.Values[i].Type = TypeString
			continue
		}
		if !KnownType(v.Type) {
			return nil, "", fmt.Errorf("value %q has unknown type %q (want one of string, int, bool, list, map, object)", v.Name, v.Type)
		}
	}

	return fm.Values, templatePart, nil
}

// ParseValuesFile parses a values document (YAML, of which JSON is a subset) into a
// map keyed by template value name. Unknown keys are rejected against defs so a typo
// cannot silently fall back to a default.
func ParseValuesFile(content string, defs []ValueDef, source string) (map[string]interface{}, error) {
	var raw map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("error parsing values file %s: %w", source, err)
	}

	known := make(map[string]bool, len(defs))
	for _, d := range defs {
		known[d.Name] = true
	}

	values := make(map[string]interface{}, len(raw))
	for k := range raw {
		if !known[k] {
			return nil, fmt.Errorf("values file %s contains unknown value %q: the template declares no such value", source, k)
		}
		values[k] = raw[k]
	}
	return values, nil
}
