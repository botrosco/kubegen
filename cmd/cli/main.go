package main

import (
	"flag"
	"log"
	"os"
	"kubegen"
)

func main() {
	// define cli flags
	image := flag.string("image", "nginx:latest", "container image to use")
	name := flag.string("name", "my-app", "name of the kubernetes resources")
	namespace := flag.string("namespace", "default", "kubernetes namespace")
	targetport := flag.int("target-port", 8080, "target port for the application")
	createservice := flag.bool("create-service", false, "whether to generate a service manifest")
	createhttproute := flag.bool("create-httproute", false, "whether to generate an httproute manifest")

	flag.parse()

	// use the exported struct from your package
	config := generator.config{
		image:           *image,
		name:            *name,
		namespace:       *namespace,
		targetport:      *targetport,
		createservice:   *createservice,
		createhttproute: *createhttproute,
	}

	// call the exported function, passing os.stdout so it prints to the terminal
	err := generator.generate(config, os.stdout)
	if err != nil {
		log.fatalf("error generating manifests: %v\n", err)
	}
}
