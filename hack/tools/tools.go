/*
Package tools is used to track binary dependencies with go modules
https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
*/
package tools

// +build tools

import (
	// linter(s)
	_ "golang.org/x/lint"
	_ "honnef.co/go/tools/cmd/staticcheck"

	// kubernetes code generators
	_ "k8s.io/code-generator/cmd/conversion-gen"
	_ "k8s.io/code-generator/cmd/deepcopy-gen"
	_ "k8s.io/code-generator/cmd/defaulter-gen"
)
