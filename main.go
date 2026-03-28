package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"text/template"

	"gopkg.in/yaml.v3"
)

//go:embed templates/*.gotmpl
var embeddedTemplates embed.FS

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

func main() {
	if len(os.Args) < 3 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	templateName := os.Args[2]

	if command != "info" && command != "generate" {
		printUsage()
		os.Exit(1)
	}

	// 1. Get template content (either external or embedded)
	content, err := getTemplateContent(templateName)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// 2. Split on the first document separator
	parts := strings.SplitN(content, "\n---\n", 2)
	if len(parts) < 2 {
		fmt.Println("Error: Invalid template format. Make sure it contains a '\\n---\\n' separator.")
		os.Exit(1)
	}

	yamlPart := parts[0]
	templatePart := parts[1]

	// 3. Parse the YAML frontmatter
	var fm Frontmatter
	err = yaml.Unmarshal([]byte(yamlPart), &fm)
	if err != nil {
		fmt.Printf("Error parsing YAML: %v\n", err)
		os.Exit(1)
	}

	// 4. Handle Commands
	if command == "info" {
		printInfo(fm.Values)
	} else if command == "generate" {
		generateManifests(fm.Values, templatePart, os.Args[3:])
	}
}

// getTemplateContent checks the local disk first, then falls back to embedded templates
func getTemplateContent(name string) (string, error) {
	// Try reading from the local filesystem first (external template)
	content, err := os.ReadFile(name)
	if err == nil {
		return string(content), nil
	}

	// If it fails, see if they passed a shorthand name for an embedded template
	// e.g., passing "deployment" maps to "templates/deployment.gotmpl"
	embeddedPath := fmt.Sprintf("templates/%s.gotmpl", name)
	content, err = embeddedTemplates.ReadFile(embeddedPath)
	if err == nil {
		return string(content), nil
	}

	// Try the exact name in case they included the .gotmpl extension in the embedded name
	embeddedPathExact := fmt.Sprintf("templates/%s", name)
	content, err = embeddedTemplates.ReadFile(embeddedPathExact)
	if err == nil {
		return string(content), nil
	}

	return "", fmt.Errorf("template '%s' not found locally or in embedded templates", name)
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  kubegen info <template-file-or-embedded-name>")
	fmt.Println("  kubegen generate <template-file-or-embedded-name> [flags...]")
}

func printInfo(values []ValueDef) {
	fmt.Println("Available Values:")
	fmt.Println(strings.Repeat("-", 80))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tTYPE\tREQUIRED\tDEFAULT\tDESCRIPTION")

	for _, v := range values {
		def := v.Default
		if def == nil {
			def = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%t\t%v\t%s\n", v.Name, v.Type, v.Required, def, v.Description)
	}
	w.Flush()
}

func generateManifests(values []ValueDef, tmplStr string, args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)

	stringFlags := make(map[string]*string)
	boolFlags := make(map[string]*bool)
	intFlags := make(map[string]*int)

	for _, v := range values {
		switch v.Type {
		case "string":
			def := ""
			if v.Default != nil {
				def = fmt.Sprintf("%v", v.Default)
			}
			stringFlags[v.Name] = fs.String(v.Name, def, v.Description)
		case "bool":
			def := false
			if v.Default != nil {
				if val, ok := v.Default.(bool); ok {
					def = val
				}
			}
			boolFlags[v.Name] = fs.Bool(v.Name, def, v.Description)
		case "int":
			def := 0
			if v.Default != nil {
				switch val := v.Default.(type) {
				case int:
					def = val
				case float64:
					def = int(val)
				}
			}
			intFlags[v.Name] = fs.Int(v.Name, def, v.Description)
		}
	}

	err := fs.Parse(args)
	if err != nil {
		fmt.Println("Error parsing flags:", err)
		os.Exit(1)
	}

	providedFlags := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		providedFlags[f.Name] = true
	})

	data := make(map[string]interface{})
	for _, v := range values {
		if v.Required && !providedFlags[v.Name] {
			fmt.Printf("Error: required flag --%s is missing\n", v.Name)
			os.Exit(1)
		}

		switch v.Type {
		case "string":
			data[v.Name] = *stringFlags[v.Name]
		case "bool":
			data[v.Name] = *boolFlags[v.Name]
		case "int":
			data[v.Name] = *intFlags[v.Name]
		}
	}

	tmpl, err := template.New("manifest").Parse(tmplStr)
	if err != nil {
		fmt.Printf("Error parsing template: %v\n", err)
		os.Exit(1)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		fmt.Printf("Error executing template: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(buf.String())
}
