package k8s

type resourceTemplate struct {
	templatePath string
	variables    map[string]string
}

type projectTemplate struct {
	image     string
	resources map[string]resourceTemplate
}

var projectTemplateConfig = map[string]projectTemplate{
	"cpp": {
		image: "ghcr.io/mudit06mah/shell-cpp:latest",
		resources: map[string]resourceTemplate{
			"shellPod": {
				templatePath: "./manifests/shell-pod.yaml",
				"variables": map[string]string{
					"SHELL_IMAGE": "ghcr.io/mudit06mah/shell-cpp:latest",
				},
			},
		},
	},
	"python": {
		image: "ghcr.io/mudit06mah/shell-python:latest",
		resources: {
			shellPod: {
				templatePath: "./manifests/shell-pod.yaml",
				variables: {
					"SHELL_IMAGE": "ghcr.io/mudit06mah/shell-python:latest",
				},
			},
		},
	},
	"nodejs": {
		image: "ghcr.io/mudit06mah/shell-nodejs:latest",
		resources: {
			shellPod: {
				templatePath: "./manifests/shell-pod.yaml",
				variables: {
					"SHELL_IMAGE": "ghcr.io/mudit06mah/shell-nodejs:latest",
				},
			},
		},
	},
	"golang": {
		image: "ghcr.io/mudit06mah/shell-golang:latest",
		resources: {
			shellPod: {
				templatePath: "./manifests/shell-pod.yaml",
				variables: {
					"SHELL_IMAGE": "ghcr.io/mudit06mah/shell-golang:latest",
				},
			},
		},
	},
	"react": {
		image: "ghcr.io/mudit06mah/shell-react:latest",
		resources: {
			shellPod: {
				templatePath: "./manifests/shell-pod.yaml",
				variables: {
					"SHELL_IMAGE": "ghcr.io/mudit06mah/shell-react:latest",
				},
			},
			service: {
				templatePath: "./manifests/service.yaml",
				variables:    {},
			},
			ingress: {
				templatePath: "./manifests/ingress.yaml",
				variables: {
					"HOST": "",
				},
			},
		},
	},
}
