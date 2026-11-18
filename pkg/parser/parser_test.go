package parser

import "testing"

func TestParseTypes(t *testing.T) {
	content := `values:
  - name: Name
    type: string
    required: true
  - name: Image
    type: string
  - name: Port
    type: int
    default: 8080
  - name: Debug
    type: bool
    default: true
  - name: Hostnames
    type: list
  - name: Env
    type: map
  - name: Object
    type: object
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Name }}
`
	values, body, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(values) != 7 {
		t.Fatalf("expected 7 values, got %d", len(values))
	}
	want := map[string]string{
		"Name": "string", "Image": "string", "Port": "int", "Debug": "bool",
		"Hostnames": "list", "Env": "map", "Object": "object",
	}
	for _, v := range values {
		if got := want[v.Name]; got != v.Type {
			t.Errorf("value %s: expected type %q, got %q", v.Name, got, v.Type)
		}
	}
	if body == "" {
		t.Error("expected a template body")
	}
}

func TestParseScalarClassification(t *testing.T) {
	defs := []ValueDef{
		{Name: "a", Type: TypeString},
		{Name: "b", Type: TypeInt},
		{Name: "c", Type: TypeBool},
		{Name: "d", Type: TypeList},
		{Name: "e", Type: TypeMap},
		{Name: "f", Type: TypeObject},
	}
	for _, d := range defs {
		wantScalar := d.Type == TypeString || d.Type == TypeInt || d.Type == TypeBool
		if d.IsScalar() != wantScalar {
			t.Errorf("%s (type %s): IsScalar() = %v, want %v", d.Name, d.Type, d.IsScalar(), wantScalar)
		}
	}
}

func TestParseMissingSeparator(t *testing.T) {
	if _, _, err := Parse("values:\n  - name: Name\n"); err == nil {
		t.Fatal("expected an error for a template with no separator")
	}
}

func TestParseUnknownType(t *testing.T) {
	content := "values:\n  - name: Name\n    type: blob\n---\nbody\n"
	if _, _, err := Parse(content); err == nil {
		t.Fatal("expected an error for an unknown value type")
	}
}

func TestParseDefaultsTypeToString(t *testing.T) {
	content := "values:\n  - name: Name\n---\nbody\n"
	values, _, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if values[0].Type != TypeString {
		t.Errorf("expected an unset type to default to string, got %q", values[0].Type)
	}
}

func TestParseValuesFile(t *testing.T) {
	defs := []ValueDef{{Name: "Name", Type: TypeString}, {Name: "Hostnames", Type: TypeList}}

	got, err := ParseValuesFile("Name: demo\nHostnames: [a, b]\n", defs, "values.yaml")
	if err != nil {
		t.Fatalf("ParseValuesFile returned an error: %v", err)
	}
	if got["Name"] != "demo" {
		t.Errorf("Name = %v, want demo", got["Name"])
	}
	hosts, ok := got["Hostnames"].([]interface{})
	if !ok || len(hosts) != 2 {
		t.Errorf("Hostnames = %v, want a two-element list", got["Hostnames"])
	}
}

func TestParseValuesFileRejectsUnknownKey(t *testing.T) {
	defs := []ValueDef{{Name: "Name", Type: TypeString}}
	_, err := ParseValuesFile("Hosntames: [a]\n", defs, "values.yaml")
	if err == nil {
		t.Fatal("expected an unknown-key error")
	}
}

func TestParseValuesFileRejectsBadYAML(t *testing.T) {
	defs := []ValueDef{{Name: "Name", Type: TypeString}}
	if _, err := ParseValuesFile("Name: [unclosed\n", defs, "values.yaml"); err == nil {
		t.Fatal("expected a parse error")
	}
}
