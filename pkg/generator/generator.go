package generator

import (
	"embed"
	"fmt"
	"io"
	"text/template"
)

//go:embed templates/manifests.gotmpl
var templateFS embed.FS

// Config holds the user inputs for the manifests.
type Config struct {
	Image           string
	Name            string
	Namespace       string
	TargetPort      int
	CreateHTTPRoute bool
	CreateSecret    bool
	ContainerPath   string
	UseGpu          bool
	CreateLiveness  bool
}

// Generate executes the embedded template with the provided config.
func Generate(cfg Config, w io.Writer) error {
	tmpl, err := template.ParseFS(templateFS, "templates/manifests.gotmpl")
	if err != nil {
		return fmt.Errorf("failed to parse embedded template: %w", err)
	}

	err = tmpl.Execute(w, cfg)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
