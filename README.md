Kubegen is a tool to simplify and speed up kubernetes workload bootstraping allowing you to enter minimal
inputs to get manifest(s) to deploy your application and its associated resources.

With kubegen you can use different templates, currently this consists of "default" and "external", these
are created in the ./pkg/generator/templates/ direcotry, these use gotmpl for structuring the data in a
similar way to helm templates.

## Usage of kubegen:
```
  -template string
        Use "default" or "external" template (defaults to "default")
```
### Template: Default
#### Required:
```
  -image string
        Container image to use
  -name string
        Name of the Kubernetes resources
  -namespace string
        Kubernetes namespace
  -target-port int
        Target port for the application
```
#### Optional
```
  -container-path string
        Path to mount the config volume (leave empty to disable volume mount)
  -create-httproute
        Generate a HTTPRoute manifest
  -create-liveness
        Add a liveness probe
  -create-secret
        Create a Secret and use it as an environment variable
  -use-gpu
        Add Intel GPU resource
```
### Template: External
#### Required:
```
  -name string
        Name of the Kubernetes resources
  -namespace string
        Kubernetes namespace
  -target-port int
        Target port for the application
```
#### Optional
```
```
