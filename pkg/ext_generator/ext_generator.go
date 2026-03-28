package ext_generator

import (
	"embed"
	"fmt"
	"io"
	"text/template"
)

//go:embed templates/external-manifests.gotmpl
var templateFS embed.FS

// Config holds the user inputs for the manifests.
type Config struct {
	Name       string
	Namespace  string
	TargetPort int
}

// Generate executes the embedded template with the provided config.
func Generate(cfg Config, w io.Writer) error {
	tmpl, err := template.ParseFS(templateFS, "templates/external-manifests.gotmpl")
	if err != nil {
		return fmt.Errorf("failed to parse embedded template: %w", err)
	}

	err = tmpl.Execute(w, cfg)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
