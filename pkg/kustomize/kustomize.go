/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package kustomize contains helpers for working with embedded kustomize commands
package kustomize

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"

	"sigs.k8s.io/kustomize/k8sdeps"
	"sigs.k8s.io/kustomize/pkg/commands/build"
	"sigs.k8s.io/kustomize/pkg/fs"
)

// PatchJSON6902 represents an inline kustomize json 6902 patch
// https://tools.ietf.org/html/rfc6902
type PatchJSON6902 struct {
	// these fields specify the patch target resource
	Group   string `json:"group"`
	Version string `json:"version"`
	Kind    string `json:"kind"`
	// Name and Namespace are optional
	// NOTE: technically name is required now, but we default it elsewhere
	// Third party users of this type / library would need to set it.
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	// Patch should contain the contents of the json patch as a string
	Patch string `json:"patch"`
}

// Build takes a set of resource blobs (yaml), patches (strategic merge patch)
// https://github.com/kubernetes/community/blob/master/contributors/devel/strategic-merge-patch.md
// and returns the `kustomize build` result as a yaml blob
// It does this in-memory using the build cobra command
func Build(resources, patches []string, patchesJSON6902 []PatchJSON6902) (string, error) {
	// write the resources and patches to an in memory fs with a generated
	// kustomization.yaml
	memFS := fs.MakeFakeFS()
	var kustomization bytes.Buffer
	fakeDir := "/"
	// for Windows we need this to be a drive because kustomize uses filepath.Abs()
	// which will add a drive letter if there is none. which drive letter is
	// unimportant as the path is on the fake filesystem anyhow
	if runtime.GOOS == "windows" {
		fakeDir = `C:\`
	}

	// NOTE: we always write this header as you cannot build without any resources
	kustomization.WriteString("resources:\n")
	for i, resource := range resources {
		// this cannot error per docs
		name := fmt.Sprintf("resource-%d.yaml", i)
		_ = memFS.WriteFile(filepath.Join(fakeDir, name), []byte(resource))
		fmt.Fprintf(&kustomization, " - %s\n", name)
	}

	if len(patches) > 0 {
		kustomization.WriteString("patches:\n")
	}
	for i, patch := range patches {
		// this cannot error per docs
		name := fmt.Sprintf("patch-%d.yaml", i)
		_ = memFS.WriteFile(filepath.Join(fakeDir, name), []byte(patch))
		fmt.Fprintf(&kustomization, " - %s\n", name)
	}

	if len(patchesJSON6902) > 0 {
		kustomization.WriteString("patchesJson6902:\n")
	}
	for i, patch := range patchesJSON6902 {
		// this cannot error per docs
		name := fmt.Sprintf("patch-json6902-%d.yaml", i)
		_ = memFS.WriteFile(filepath.Join(fakeDir, name), []byte(patch.Patch))
		fmt.Fprintf(&kustomization, " - path: %s\n", name)
		fmt.Fprintf(&kustomization, "   target:\n")
		fmt.Fprintf(&kustomization, "     group: %s\n", patch.Group)
		fmt.Fprintf(&kustomization, "     version: %s\n", patch.Version)
		fmt.Fprintf(&kustomization, "     kind: %s\n", patch.Kind)
		if patch.Name != "" {
			fmt.Fprintf(&kustomization, "     name: %s\n", patch.Name)
		}
		if patch.Namespace != "" {
			fmt.Fprintf(&kustomization, "     namespace: %s\n", patch.Namespace)
		}
	}

	memFS.WriteFile(filepath.Join(fakeDir, "kustomization.yaml"), kustomization.Bytes())

	// now we can build the kustomization
	var out bytes.Buffer
	f := k8sdeps.NewFactory()
	cmd := build.NewCmdBuild(&out, memFS, f.ResmapF, f.TransformerF)
	cmd.SetArgs([]string{fakeDir})
	// we want to silence usage, error output, and any future output from cobra
	// we will get error output as a golang error from execute
	cmd.SetOutput(ioutil.Discard)
	_, err := cmd.ExecuteC()
	if err != nil {
		return "", err
	}
	return out.String(), nil
}
