package main

import (
	"flag"
	"fmt"
	"kubegen/pkg/generator"
	"log"
	"os"
)

func main() {
	image := flag.String("image", "", "Container image to use (Required)")
	name := flag.String("name", "", "Name of the Kubernetes resources (Required)")
	namespace := flag.String("namespace", "", "Kubernetes namespace (Required)")

	targetPort := flag.Int("target-port", 0, "Target port for the application (Required)")
	createHTTPRoute := flag.Bool("create-httproute", false, "Generate an HTTPRoute manifest")
	createSecret := flag.Bool("create-secret", false, "Create a Secret and inject it as an environment variable")
	containerPath := flag.String("container-path", "", "Path to mount the config volume (leave empty to disable)")
	useGpu := flag.Bool("use-gpu", false, "Add Intel GPU resource limits")
	createLiveness := flag.Bool("create-liveness", false, "Add a liveness probe")

	flag.Parse()

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
