package generator

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"text/template"

	"kubegen/pkg/parser"

	"gopkg.in/yaml.v3"
)

// filenameAnnotation lets a document choose its output file name.
const filenameAnnotation = "kubegen.io/filename"

// Options carries the invocation inputs that are not template values.
type Options struct {
	// Args are the command-line arguments after the template name.
	Args []string
	// Stdin is read when the values flag names "-".
	Stdin io.Reader
	// Stdout receives progress messages.
	Stdout io.Writer
}

func PrintInfo(values []parser.ValueDef) {
	fmt.Println("Available Values:")
	fmt.Println(strings.Repeat("-", 80))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tTYPE\tREQUIRED\tDEFAULT\tSOURCE\tDESCRIPTION")

	for _, v := range values {
		def := v.Default
		if def == nil {
			def = "-"
		}
		source := "flag"
		if !v.IsScalar() {
			source = "values file"
		}
		fmt.Fprintf(w, "%s\t%s\t%t\t%v\t%s\t%s\n", v.Name, v.Type, v.Required, def, source, v.Description)
	}
	w.Flush()
}

// templateFuncs are the helpers templates may use. Kept small and documented in README.
var templateFuncs = template.FuncMap{
	// isZero reports an empty or zero quantity, so templates can omit what nobody chose.
	"isZero": func(v interface{}) bool {
		if v == nil {
			return true
		}
		switch t := v.(type) {
		case string:
			s := strings.TrimSpace(t)
			return s == "" || s == "0" || s == "0m" || s == "0Mi" || s == "0Gi" || s == "0M" || s == "0G"
		case int:
			return t == 0
		case float64:
			return t == 0
		case bool:
			return !t
		case []interface{}:
			return len(t) == 0
		case map[string]interface{}:
			return len(t) == 0
		case []string:
			return len(t) == 0
		}
		return false
	},
	"quote": func(v interface{}) string {
		return fmt.Sprintf("%q", fmt.Sprintf("%v", v))
	},
	// squote renders single-quoted YAML, so regexes stay readable ('^build-[0-9]+$').
	"squote": func(v interface{}) string {
		s := fmt.Sprintf("%v", v)
		return "'" + strings.ReplaceAll(s, "'", "''") + "'"
	},
	"join": strings.Join,
	// repoOnly strips a tag from an image reference, keeping any registry port:
	// "ghcr.io/a/b:1.2.3" -> "ghcr.io/a/b", "reg:5000/a" -> "reg:5000/a".
	"repoOnly": func(image string) string {
		slash := strings.LastIndex(image, "/")
		colon := strings.LastIndex(image, ":")
		if colon > slash {
			return image[:colon]
		}
		return image
	},
	// hasKey reports whether a map value was supplied, so a deliberate zero is not
	// mistaken for an absent value.
	"hasKey": func(m interface{}, key string) bool {
		mm, ok := m.(map[string]interface{})
		if !ok {
			return false
		}
		_, present := mm[key]
		return present
	},
	// fail aborts the render with a message, for template-side validation.
	"fail": func(msg string) (string, error) {
		return "", fmt.Errorf("%s", msg)
	},
	"lower": strings.ToLower,
	"default": func(def, v interface{}) interface{} {
		if v == nil || v == "" {
			return def
		}
		return v
	},
	// indent renders a nested block indented by n spaces; "" stays "".
	"indent": func(n int, s string) string {
		if s == "" {
			return ""
		}
		pad := strings.Repeat(" ", n)
		lines := strings.Split(s, "\n")
		for i, l := range lines {
			if l == "" {
				continue
			}
			lines[i] = pad + l
		}
		return strings.Join(lines, "\n")
	},
}

func GenerateManifests(values []parser.ValueDef, tmplStr string, args []string) error {
	return GenerateManifestsWith(values, tmplStr, Options{Args: args, Stdin: os.Stdin, Stdout: os.Stdout})
}

func GenerateManifestsWith(values []parser.ValueDef, tmplStr string, opts Options) error {
	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stdin := opts.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}

	// ContinueOnError so an undefined flag is a returned error rather than a process
	// exit: a list value passed as a flag must be reportable, and tests must survive it.
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)

	var outputDir string
	fs.StringVar(&outputDir, "output", "", "Output directory path (prints to stdout if omitted)")
	fs.StringVar(&outputDir, "o", "", "Shorthand for --output")

	var valuesPath string
	fs.StringVar(&valuesPath, "values", "", "Values file (YAML or JSON) supplying template values; '-' reads stdin")
	fs.StringVar(&valuesPath, "f", "", "Shorthand for --values")

	stringFlags := make(map[string]*string)
	boolFlags := make(map[string]*bool)
	intFlags := make(map[string]*int)

	for _, v := range values {
		if strings.ToLower(v.Name) == "output" || strings.ToLower(v.Name) == "o" {
			return fmt.Errorf("template variable cannot be named '%s' as it is a reserved flag", v.Name)
		}
		if strings.ToLower(v.Name) == "values" || strings.ToLower(v.Name) == "f" {
			return fmt.Errorf("template variable cannot be named '%s' as it is a reserved flag", v.Name)
		}

		switch v.Type {
		case parser.TypeString:
			def := ""
			if v.Default != nil {
				def = fmt.Sprintf("%v", v.Default)
			}
			stringFlags[v.Name] = fs.String(v.Name, def, v.Description)
		case parser.TypeBool:
			def := false
			if v.Default != nil {
				if val, ok := v.Default.(bool); ok {
					def = val
				}
			}
			boolFlags[v.Name] = fs.Bool(v.Name, def, v.Description)
		case parser.TypeInt:
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

	if err := fs.Parse(opts.Args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return fmt.Errorf("error parsing flags: %w", err)
	}

	providedFlags := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		providedFlags[f.Name] = true
	})

	// Values file: loaded before the merge so it sits between defaults and flags.
	fileValues := map[string]interface{}{}
	if valuesPath != "" {
		var content []byte
		var err error
		source := valuesPath
		if valuesPath == "-" {
			source = "stdin"
			content, err = io.ReadAll(stdin)
		} else {
			content, err = os.ReadFile(valuesPath)
		}
		if err != nil {
			return fmt.Errorf("failed to read values file %s: %w", source, err)
		}
		fileValues, err = parser.ParseValuesFile(string(content), values, source)
		if err != nil {
			return err
		}
	}

	data := make(map[string]interface{})
	for _, v := range values {
		var val interface{}
		var have bool

		// 1. frontmatter default
		if v.Default != nil {
			val, have = v.Default, true
		}
		// 2. values file
		if fv, ok := fileValues[v.Name]; ok {
			val, have = fv, true
		}
		// 3. explicit flag (scalars only)
		if v.IsScalar() && providedFlags[v.Name] {
			switch v.Type {
			case parser.TypeString:
				val = *stringFlags[v.Name]
			case parser.TypeBool:
				val = *boolFlags[v.Name]
			case parser.TypeInt:
				val = *intFlags[v.Name]
			}
			have = true
		}

		if v.Required && !have {
			if v.IsScalar() {
				return fmt.Errorf("required flag --%s is missing (or supply it in a values file)", v.Name)
			}
			return fmt.Errorf("required value %q is missing: supply it in a values file with -f", v.Name)
		}

		coerced, err := coerce(v, val)
		if err != nil {
			return err
		}
		data[v.Name] = coerced
	}

	tmpl, err := template.New("manifest").Funcs(templateFuncs).Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("error parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("error executing template: %w", err)
	}

	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
		}

		docs := splitDocuments(buf.String())

		fmt.Fprintf(stdout, "Writing manifests to directory: %s\n", outputDir)

		for _, doc := range docs {
			var resource map[string]interface{}
			if err := yaml.Unmarshal([]byte(doc), &resource); err != nil {
				return fmt.Errorf("failed to parse generated YAML chunk: %w\n\n%s", err, doc)
			}

			kind, _ := resource["kind"].(string)
			if kind == "" {
				kind = "Unknown"
			}

			var name string
			if metadata, ok := resource["metadata"].(map[string]interface{}); ok {
				name, _ = metadata["name"].(string)
			}
			if name == "" {
				name = "unnamed"
			}

			// Uniform naming for every kind: <name>-<kind>.yaml, unless the document
			// names its own file (a kustomization.yaml must not be <name>-kustomization.yaml).
			fileName := fmt.Sprintf("%s-%s.yaml", name, strings.ToLower(kind))
			if override, ok := filenameOverride(resource); ok {
				fileName = override
				doc = stripFilenameAnnotation(doc)
			}

			filePath := filepath.Join(outputDir, fileName)

			outData := []byte(fmt.Sprintf("---\n%s\n", doc))
			if err := os.WriteFile(filePath, outData, 0644); err != nil {
				return fmt.Errorf("failed to write file %s: %w", filePath, err)
			}

			fmt.Fprintf(stdout, "  - Created: %s\n", fileName)
		}
	} else {
		fmt.Fprintln(stdout, buf.String())
	}

	return nil
}

// coerce normalizes a merged value to the type the template expects.
func coerce(v parser.ValueDef, val interface{}) (interface{}, error) {
	switch v.Type {
	case parser.TypeString:
		if val == nil {
			return "", nil
		}
		return fmt.Sprintf("%v", val), nil
	case parser.TypeInt:
		switch t := val.(type) {
		case nil:
			return 0, nil
		case int:
			return t, nil
		case int64:
			return int(t), nil
		case float64:
			return int(t), nil
		case string:
			var n int
			if _, err := fmt.Sscanf(t, "%d", &n); err != nil {
				return nil, fmt.Errorf("value %q: cannot use %q as int", v.Name, t)
			}
			return n, nil
		}
		return nil, fmt.Errorf("value %q: cannot use %v as int", v.Name, val)
	case parser.TypeBool:
		switch t := val.(type) {
		case nil:
			return false, nil
		case bool:
			return t, nil
		case string:
			switch strings.ToLower(strings.TrimSpace(t)) {
			case "true", "yes", "1":
				return true, nil
			case "false", "no", "0", "":
				return false, nil
			}
			return nil, fmt.Errorf("value %q: cannot use %q as bool", v.Name, t)
		}
		return nil, fmt.Errorf("value %q: cannot use %v as bool", v.Name, val)
	case parser.TypeList:
		switch t := val.(type) {
		case nil:
			return []interface{}{}, nil
		case []interface{}:
			return t, nil
		}
		return nil, fmt.Errorf("value %q: expected a list, got %T", v.Name, val)
	case parser.TypeMap, parser.TypeObject:
		switch t := val.(type) {
		case nil:
			return map[string]interface{}{}, nil
		case map[string]interface{}:
			return t, nil
		case map[interface{}]interface{}:
			out := make(map[string]interface{}, len(t))
			for k, v := range t {
				out[fmt.Sprintf("%v", k)] = v
			}
			return out, nil
		}
		return nil, fmt.Errorf("value %q: expected a map, got %T", v.Name, val)
	}
	return val, nil
}

// filenameOverride reads and removes the kubegen.io/filename annotation, which lets a
// template choose its own file name where a convention demands one (kustomization.yaml).
func filenameOverride(resource map[string]interface{}) (string, bool) {
	metadata, ok := resource["metadata"].(map[string]interface{})
	if !ok {
		return "", false
	}
	annotations, ok := metadata["annotations"].(map[string]interface{})
	if !ok {
		return "", false
	}
	name, ok := annotations[filenameAnnotation].(string)
	if !ok || name == "" {
		return "", false
	}
	delete(annotations, filenameAnnotation)
	if len(annotations) == 0 {
		delete(metadata, "annotations")
	}
	return name, true
}

// stripFilenameAnnotation removes the kubegen.io/filename annotation from the raw
// document text, so the written file does not carry kubegen's own bookkeeping. It works
// line-wise to keep every other line (comments included) byte-identical, and drops the
// annotations key if the filename entry was its only content.
func stripFilenameAnnotation(doc string) string {
	lines := strings.Split(doc, "\n")
	out := make([]string, 0, len(lines))
	annotationsIdx := -1

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, filenameAnnotation+":") {
			continue
		}
		if trimmed == "annotations:" {
			annotationsIdx = len(out)
		}
		out = append(out, line)
	}

	if annotationsIdx < 0 {
		return strings.Join(out, "\n")
	}

	indent := func(s string) int { return len(s) - len(strings.TrimSpace(s)) }
	// Keep the key only if some following line is nested under it.
	for _, line := range out[annotationsIdx+1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if indent(line) > indent(out[annotationsIdx]) {
			return strings.Join(out, "\n")
		}
		break
	}
	out = append(out[:annotationsIdx], out[annotationsIdx+1:]...)
	return strings.Join(out, "\n")
}

// splitDocuments splits a rendered stream into documents on lines that are exactly a
// document separator. A "---" inside an indented block scalar is not a separator,
// and comments are preserved because the raw text is kept.
func splitDocuments(rendered string) []string {
	lines := strings.Split(rendered, "\n")
	var docs []string
	var current []string

	flush := func() {
		doc := strings.TrimSpace(strings.Join(current, "\n"))
		if doc != "" {
			docs = append(docs, doc)
		}
		current = nil
	}

	for _, line := range lines {
		if strings.TrimRight(line, " \t") == "---" {
			flush()
			continue
		}
		current = append(current, line)
	}
	flush()
	return docs
}

// SortedKeys is exported for templates that need deterministic map iteration.
func SortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
