package k8s

import (
	"fmt"
	"os"
	"strings"
)

type ResourceTemplate struct {
	templatePath string
	variables    map[string]string
}

type ProjectTemplate struct {
	image     string
	resources map[string]ResourceTemplate
}

var ProjectTemplateConfig = map[string]ProjectTemplate{
	"cpp": {
		image: "ghcr.io/mudit06mah/shell-cpp:latest",
		resources: map[string]ResourceTemplate{
			"shellPod": {
				templatePath: "./manifests/shell-pod.yaml",
				variables: map[string]string{
					"SHELL_IMAGE": "ghcr.io/mudit06mah/shell-cpp:latest",
				},
			},
		},
	},
	"python": {
		image: "ghcr.io/mudit06mah/shell-python:latest",
		resources: map[string]ResourceTemplate{
			"shellPod": {
				templatePath: "./manifests/shell-pod.yaml",
				variables: map[string]string{
					"SHELL_IMAGE": "ghcr.io/mudit06mah/shell-python:latest",
				},
			},
		},
	},
	"nodejs": {
		image: "ghcr.io/mudit06mah/shell-nodejs:latest",
		resources: map[string]ResourceTemplate{
			"shellPod": {
				templatePath: "./manifests/shell-pod.yaml",
				variables: map[string]string{
					"SHELL_IMAGE": "ghcr.io/mudit06mah/shell-nodejs:latest",
				},
			},
		},
	},
	"golang": {
		image: "ghcr.io/mudit06mah/shell-golang:latest",
		resources: map[string]ResourceTemplate{
			"shellPod": {
				templatePath: "./manifests/shell-pod.yaml",
				variables: map[string]string{
					"SHELL_IMAGE": "ghcr.io/mudit06mah/shell-golang:latest",
				},
			},
		},
	},
	"react": {
		image: "ghcr.io/mudit06mah/shell-react:latest",
		resources: map[string]ResourceTemplate{
			"shellPod": {
				templatePath: "./manifests/shell-pod.yaml",
				variables: map[string]string{
					"SHELL_IMAGE": "ghcr.io/mudit06mah/shell-react:latest",
				},
			},
			"service": {
				templatePath: "./manifests/service.yaml",
				variables:    map[string]string{},
			},
			"ingress": {
				templatePath: "./manifests/ingress.yaml",
				variables: map[string]string{
					"HOST": "",
				},
			},
		},
	},
}

func getProjectConfig(projectType string) (ProjectTemplate, error) {
	config, exists := ProjectTemplateConfig[projectType]
	if !exists {
		return ProjectTemplate{}, fmt.Errorf("Unsupported project type: %s", projectType)
	}
	return config, nil
}

func (c *Client) RenderProjectResources(projectType string) ([][]byte, error) {
	projectTemplate, err := getProjectConfig(projectType)

	if err != nil {
		return nil, err
	}

	var commonVars = map[string]string{
		"WORKSPACE_ID": workspaceId,
		"NAMESPACE":    namespace,
	}

	var manifestRender [][]byte

	for resourceName, resourceTemplate := range projectTemplate.resources {
		allVars := make(map[string]string)

		for k, v := range commonVars {
			allVars[k] = v
		}

		for k, v := range resourceTemplate.variables {
			allVars[k] = v
		}

		// Dynamic Values (todo: recheck this)
		if resourceName == "ingress" {
			allVars["HOST"] = workspaceId+".localhost"
		}

		manifest, err := RenderTemplate(resourceTemplate.templatePath, allVars)
		if err != nil {
			return nil, err
		}

		manifestRender = append(manifestRender, manifest)

	}

	return manifestRender, nil
}

func RenderTemplate(templatePath string, replace map[string]string) ([]byte, error) {
	yaml, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %v", err)
	}
	for key, value := range replace {
		yaml = []byte(strings.ReplaceAll(string(yaml), "{{"+key+"}}", value))
	}

	return yaml, nil
}
