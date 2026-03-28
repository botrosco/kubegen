package parser

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
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

	return fm.Values, templatePart, nil
}
