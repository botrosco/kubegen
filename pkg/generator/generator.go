package generator

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"text/template"

	"kubegen/pkg/parser"
)

func PrintInfo(values []parser.ValueDef) {
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

func GenerateManifests(values []parser.ValueDef, tmplStr string, args []string) error {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)

	stringFlags := make(map[string]*string)
	boolFlags := make(map[string]*bool)
	intFlags := make(map[string]*int)

	// Dynamically register flags
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

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("error parsing flags: %w", err)
	}

	providedFlags := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		providedFlags[f.Name] = true
	})

	data := make(map[string]interface{})
	for _, v := range values {
		if v.Required && !providedFlags[v.Name] {
			return fmt.Errorf("required flag --%s is missing", v.Name)
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
		return fmt.Errorf("error parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("error executing template: %w", err)
	}

	fmt.Println(buf.String())
	return nil
}
