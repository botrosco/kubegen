package main

import (
	"flag"
	"fmt"
	"kubegen/pkg/ext_generator"
	"kubegen/pkg/generator"
	"log"
	"os"
)

func main() {
	template := flag.String("template", "default", "Use \"default\" or \"external\" template")

	name := flag.String("name", "", "Name of the Kubernetes resources (Required) [type:default,external]")
	namespace := flag.String("namespace", "", "Kubernetes namespace (Required) [type:default,external]")
	targetPort := flag.Int("target-port", 0, "Target port for the application (Required) [type:default,external]")
	image := flag.String("image", "", "Container image to use (Required) [type:default]")
	createHTTPRoute := flag.Bool("create-httproute", false, "Generate a HTTPRoute manifest [type:default]")
	createSecret := flag.Bool("create-secret", false, "Create a Secret and inject it as an environment variable [type:default]")
	containerPath := flag.String("container-path", "", "Path to mount the config volume (leave empty to disable) [type:default]")
	useGpu := flag.Bool("use-gpu", false, "Add Intel GPU resource limits [type:default]")
	createLiveness := flag.Bool("create-liveness", false, "Add a liveness probe [type:default]")
	flag.Parse()

	if *template == "external" {
		if *name == "" || *namespace == "" || *targetPort == 0 {
			fmt.Println("Error: Missing required flags.")
			fmt.Println("You must provide -name, -namespace and -target-port.")
			fmt.Println()
			flag.Usage()
			os.Exit(1)
		}

		// Build the config struct
		config := ext_generator.Config{
			Name:       *name,
			Namespace:  *namespace,
			TargetPort: *targetPort,
		}

		// Run the generator
		err := ext_generator.Generate(config, os.Stdout)
		if err != nil {
			log.Fatalf("Error generating manifests: %v\n", err)
		}
	} else {
		if *name == "" || *namespace == "" || *image == "" {
			fmt.Println("Error: Missing required flags.")
			fmt.Println("You must provide -name, -namespace, -image, and -target-port.")
			fmt.Println()
			flag.Usage()
			os.Exit(1)
		}

		// Build the config struct
		config := generator.Config{
			Image:           *image,
			Name:            *name,
			Namespace:       *namespace,
			TargetPort:      *targetPort,
			CreateHTTPRoute: *createHTTPRoute,
			CreateSecret:    *createSecret,
			ContainerPath:   *containerPath,
			UseGpu:          *useGpu,
			CreateLiveness:  *createLiveness,
		}

		// Run the generator
		err := generator.Generate(config, os.Stdout)
		if err != nil {
			log.Fatalf("Error generating manifests: %v\n", err)
		}
	}
}
