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
	"fmt"
	"path/filepath"
	"runtime"

	"sigs.k8s.io/kustomize/v3/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/v3/k8sdeps/transformer"
	"sigs.k8s.io/kustomize/v3/pkg/fs"
	"sigs.k8s.io/kustomize/v3/pkg/gvk"
	"sigs.k8s.io/kustomize/v3/pkg/loader"
	"sigs.k8s.io/kustomize/v3/pkg/plugins"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/resource"
	"sigs.k8s.io/kustomize/v3/pkg/target"
	"sigs.k8s.io/kustomize/v3/pkg/types"
	"sigs.k8s.io/kustomize/v3/pkg/validators"
	"sigs.k8s.io/kustomize/v3/plugin/builtin"
	"sigs.k8s.io/yaml"

	"sigs.k8s.io/kind/pkg/errors"

	"sigs.k8s.io/kind/pkg/internal/apis/config"
)

// Build takes a set of resource blobs (yaml), patches (strategic merge patch)
// https://github.com/kubernetes/community/blob/master/contributors/devel/strategic-merge-patch.md
// and returns the `kustomize build` result as a yaml blob
// It does this in-memory using the build cobra command
func Build(resources, patches []string, patchesJSON6902 []config.PatchJSON6902) (string, error) {
	// write the resources and patches to an in memory fs with a generated
	// kustomization.yaml
	memFS := fs.MakeFakeFS() // fs.MakeFsInMemory()
	fakeDir := "/"
	// for Windows we need this to be a drive because kustomize uses filepath.Abs()
	// which will add a drive letter if there is none. which drive letter is
	// unimportant as the path is on the fake filesystem anyhow
	if runtime.GOOS == "windows" {
		fakeDir = `C:\`
	}

	// NOTE: we always write this header as you cannot build without any resources
	kustomization := &types.Kustomization{
		TypeMeta: types.TypeMeta{
			APIVersion: types.KustomizationVersion,
			Kind:       types.KustomizationKind,
		},
		PatchesStrategicMerge: make([]types.PatchStrategicMerge, 0, len(patches)),
		PatchesJson6902:       make([]types.PatchJson6902, 0, len(patchesJSON6902)),
		Resources:             make([]string, 0, len(resources)),
	}

	for i, resource := range resources {
		// this cannot error per docs
		name := fmt.Sprintf("resource-%d.yaml", i)
		_ = memFS.WriteFile(filepath.Join(fakeDir, name), []byte(resource))
		kustomization.Resources = append(kustomization.Resources, name)
	}

	for i, patch := range patches {
		// this cannot error per docs
		name := fmt.Sprintf("patch-%d.yaml", i)
		_ = memFS.WriteFile(filepath.Join(fakeDir, name), []byte(patch))
		kustomization.PatchesStrategicMerge = append(kustomization.PatchesStrategicMerge, types.PatchStrategicMerge(name))
	}

	for i, patch := range patchesJSON6902 {
		// this cannot error per docs
		name := fmt.Sprintf("patch-json6902-%d.yaml", i)
		_ = memFS.WriteFile(filepath.Join(fakeDir, name), []byte(patch.Patch))
		kustomization.PatchesJson6902 = append(kustomization.PatchesJson6902, types.PatchJson6902{
			Path: name,
			Target: &types.PatchTarget{
				Gvk: gvk.Gvk{
					Group:   patch.Group,
					Version: patch.Version,
				},
				Name:      patch.Name,
				Namespace: patch.Namespace,
			},
		})
	}

	// Write out kustomization.yaml
	buf, err := yaml.Marshal(kustomization)
	if err != nil {
		return "", errors.Wrap(err, "error marshaling kustomization.yaml")
	}
	if err := memFS.WriteFile(filepath.Join(fakeDir, "kustomization.yaml"), buf); err != nil {
		return "", errors.Wrap(err, "error writing kustomization.yaml to memFS")
	}

	// now we can build the kustomization
	// Note; kustomize has internalized cmd build kustomize 3.2.0
	lrc := loader.RestrictionRootOnly
	ldr, err := loader.NewLoader(lrc, validators.MakeFakeValidator(), fakeDir, memFS)
	if err != nil {
		return "", errors.Wrap(err, "error executing kustomize build")
	}
	defer func() {
		_ = ldr.Cleanup()
	}()
	rf := resmap.NewFactory(resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl()), transformer.NewFactoryImpl())
	pc := plugins.DefaultPluginConfig()
	kt, err := target.NewKustTarget(ldr, rf, transformer.NewFactoryImpl(), plugins.NewLoader(pc, rf))
	if err != nil {
		return "", errors.Wrap(err, "error executing kustomize build")
	}
	allResources, err := kt.MakeCustomizedResMap()
	if err != nil {
		return "", errors.Wrap(err, "error executing kustomize build")
	}
	err = builtin.NewLegacyOrderTransformerPlugin().Transform(allResources)
	if err != nil {
		return "", errors.Wrap(err, "error executing kustomize build")
	}
	yamlResources, err := allResources.AsYaml()
	if err != nil {
		return "", errors.Wrap(err, "error executing kustomize build")
	}
	return string(yamlResources), nil
}
