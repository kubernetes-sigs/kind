module sigs.k8s.io/kind/hack/tools

go 1.13

require (
	github.com/golangci/golangci-lint v1.46.2
	// TODO: remove when no longer needed to be explicit to pickup CVE fix
	// indirect dep
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gotest.tools/gotestsum v1.8.1
	k8s.io/code-generator v0.24.1
)
