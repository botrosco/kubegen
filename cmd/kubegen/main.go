package main

import (
	"fmt"
	"os"

	"kubegen/pkg/generator"
	"kubegen/pkg/parser"
	"kubegen/pkg/tpl"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	if command == "list" {
		templates, err := tpl.ListEmbedded()
		if err != nil {
			fmt.Printf("Error listing templates: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Bundled Templates:")
		for _, t := range templates {
			fmt.Printf("  - %s\n", t)
		}
		return
	}

	// Info and generate requires template name
	if len(os.Args) < 3 {
		printUsage()
		os.Exit(1)
	}

	templateName := os.Args[2]

	if command != "info" && command != "generate" {
		printUsage()
		os.Exit(1)
	}

	// Get template content
	content, err := tpl.GetContent(templateName)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Parse YAML and split template
	values, templateStr, err := parser.Parse(content)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Exec
	if command == "info" {
		generator.PrintInfo(values)
	} else if command == "generate" {
		err := generator.GenerateManifests(values, templateStr, os.Args[3:])
		if err != nil {
			fmt.Printf("Generation Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  kubegen list")
	fmt.Println("  kubegen info <template-file-or-embedded-name>")
	fmt.Println("  kubegen generate <template-file-or-embedded-name> [flags...] [-o|--output (Dir to put manifests)]")
}
