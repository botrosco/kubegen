package generator

import (
	"embed"
	"fmt"
	"io"
	"text/template"
)

//go:embed templates/manifests.tmpl
var templateFS embed.FS

// Config holds the values for the Kubernetes manifests.
// Exported fields (capitalized) can be accessed by the importing code.
type Config struct {
	Image           string
	Name            string
	Namespace       string
	TargetPort      int
	CreateService   bool
	CreateHTTPRoute bool
}

// Generate takes a Config and an io.Writer, renders the template,
// and writes the output. Returning an error allows the caller to handle it.
func Generate(cfg Config, w io.Writer) error {
	// Parse the template directly from the embedded file system
	tmpl, err := template.ParseFS(templateFS, "templates/manifests.tmpl")
	if err != nil {
		return fmt.Errorf("failed to parse embedded template: %w", err)
	}

	// Execute the template, writing to the provided io.Writer
	err = tmpl.Execute(w, cfg)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
