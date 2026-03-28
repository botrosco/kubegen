package main

import (
	"fmt"
	"os"

	"kubegen/pkg/generator"
	"kubegen/pkg/parser"
	"kubegen/pkg/tpl"
)

func main() {
	// Require at least a command
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Handle the 'list' command first, as it doesn't need a template name
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

	// For info and generate, we require the template name as well
	if len(os.Args) < 3 {
		printUsage()
		os.Exit(1)
	}

	templateName := os.Args[2]

	if command != "info" && command != "generate" {
		printUsage()
		os.Exit(1)
	}

	// 1. Get template content
	content, err := tpl.GetContent(templateName)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// 2. Parse YAML and split template
	values, templateStr, err := parser.Parse(content)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// 3. Execute requested command
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
	fmt.Println("  kubegen generate <template-file-or-embedded-name> [flags...]")
}
