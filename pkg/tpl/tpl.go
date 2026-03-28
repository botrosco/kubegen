package tpl

import (
	"embed"
	"fmt"
	"os"
	"strings"
)

//go:embed templates/*.gotmpl
var embeddedTemplates embed.FS

func GetContent(name string) (string, error) {
	// Try reading from the local filesystem first
	content, err := os.ReadFile(name)
	if err == nil {
		return string(content), nil
	}

	// Try embedded shorthand (e.g., "deployment")
	embeddedPath := fmt.Sprintf("templates/%s.gotmpl", name)
	content, err = embeddedTemplates.ReadFile(embeddedPath)
	if err == nil {
		return string(content), nil
	}

	// Try exact embedded name (e.g., "deployment.gotmpl")
	embeddedPathExact := fmt.Sprintf("templates/%s", name)
	content, err = embeddedTemplates.ReadFile(embeddedPathExact)
	if err == nil {
		return string(content), nil
	}

	return "", fmt.Errorf("template '%s' not found locally or in embedded templates", name)
}

func ListEmbedded() ([]string, error) {
	entries, err := embeddedTemplates.ReadDir("templates")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded templates directory: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".gotmpl") {
			// Strip the extension for cleaner output
			name := strings.TrimSuffix(entry.Name(), ".gotmpl")
			names = append(names, name)
		}
	}
	return names, nil
}
